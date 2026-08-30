package storage

import (
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "modernc.org/sqlite"
	"v2ray_checker/pkg/model"
)

type Store struct {
	db *sql.DB
	mu sync.RWMutex
}

func NewStore(dbPath string) (*Store, error) {
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("failed to open sqlite database: %w", err)
	}

	// Optimize SQLite for concurrent reading & writing
	if _, err := db.Exec(`PRAGMA journal_mode = WAL; PRAGMA synchronous = NORMAL;`); err != nil {
		return nil, err
	}

	createTableSQL := `
	CREATE TABLE IF NOT EXISTS nodes (
		hash TEXT PRIMARY KEY,
		config TEXT NOT NULL,
		protocol TEXT NOT NULL,
		ping TEXT NOT NULL,
		country TEXT NOT NULL,
		ip TEXT NOT NULL,
		source TEXT NOT NULL,
		is_alive INTEGER NOT NULL,
		fail_count INTEGER NOT NULL DEFAULT 0,
		first_seen_at DATETIME NOT NULL,
		last_check_at DATETIME NOT NULL
	);
	CREATE INDEX IF NOT EXISTS idx_nodes_alive ON nodes(is_alive);
	CREATE INDEX IF NOT EXISTS idx_nodes_protocol ON nodes(protocol);
	`
	if _, err := db.Exec(createTableSQL); err != nil {
		return nil, fmt.Errorf("failed to initialize schema: %w", err)
	}

	return &Store{db: db}, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) SaveTestedNode(node *model.Node) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	query := `
	INSERT INTO nodes (hash, config, protocol, ping, country, ip, source, is_alive, fail_count, first_seen_at, last_check_at)
	VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	ON CONFLICT(hash) DO UPDATE SET
		config = excluded.config,
		ping = excluded.ping,
		country = CASE WHEN excluded.country != '' THEN excluded.country ELSE nodes.country END,
		ip = CASE WHEN excluded.ip != '' THEN excluded.ip ELSE nodes.ip END,
		is_alive = excluded.is_alive,
		fail_count = CASE WHEN excluded.is_alive = 1 THEN 0 ELSE nodes.fail_count + 1 END,
		last_check_at = excluded.last_check_at;
	`
	aliveInt := 0
	if node.IsAlive {
		aliveInt = 1
	}

	now := time.Now().UTC()
	firstSeen := node.FirstSeenAt
	if firstSeen.IsZero() {
		firstSeen = now
	}

	_, err := s.db.Exec(query,
		node.Hash,
		node.Config,
		node.Protocol,
		node.Ping,
		node.Country,
		node.IP,
		node.Source,
		aliveInt,
		node.FailCount,
		firstSeen,
		now,
	)
	return err
}

func (s *Store) GetActiveNodes(protocolFilter string) ([]model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	query := `SELECT config, ping, country, ip, protocol, hash, source, is_alive, fail_count, first_seen_at, last_check_at
	          FROM nodes WHERE is_alive = 1`
	var args []any

	if protocolFilter != "" && protocolFilter != "all" {
		query += ` AND protocol = ?`
		args = append(args, protocolFilter)
	}
	query += ` ORDER BY CAST(ping AS INTEGER) ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		var isAliveInt int
		err := rows.Scan(
			&n.Config,
			&n.Ping,
			&n.Country,
			&n.IP,
			&n.Protocol,
			&n.Hash,
			&n.Source,
			&isAliveInt,
			&n.FailCount,
			&n.FirstSeenAt,
			&n.LastCheckAt,
		)
		if err != nil {
			return nil, err
		}
		n.IsAlive = (isAliveInt == 1)
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (s *Store) GetAllNodesForCheck() ([]model.Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rows, err := s.db.Query(`SELECT hash, config, protocol, source, fail_count FROM nodes`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var nodes []model.Node
	for rows.Next() {
		var n model.Node
		if err := rows.Scan(&n.Hash, &n.Config, &n.Protocol, &n.Source, &n.FailCount); err != nil {
			return nil, err
		}
		nodes = append(nodes, n)
	}
	return nodes, nil
}

func (s *Store) PruneDeadNodes(maxFailures int) (int64, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	res, err := s.db.Exec(`DELETE FROM nodes WHERE fail_count >= ? AND is_alive = 0`, maxFailures)
	if err != nil {
		return 0, err
	}
	return res.RowsAffected()
}

func (s *Store) GetStats() (map[string]any, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]any)
	var total, alive int
	_ = s.db.QueryRow(`SELECT COUNT(*), COALESCE(SUM(CASE WHEN is_alive = 1 THEN 1 ELSE 0 END), 0) FROM nodes`).Scan(&total, &alive)

	stats["total_nodes"] = total
	stats["alive_nodes"] = alive

	return stats, nil
}
