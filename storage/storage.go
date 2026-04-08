package storage

import (
	"database/sql"
	"fmt"
	"log"
	"math/rand"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	_ "github.com/mattn/go-sqlite3"

	"proxygate/config"
)

type Proxy struct {
	ID             int64     `json:"id"`
	Address        string    `json:"address"`
	Protocol       string    `json:"protocol"`
	ExitIP         string    `json:"exit_ip"`
	ExitLocation   string    `json:"exit_location"`
	Latency        int       `json:"latency"`
	QualityGrade   string    `json:"quality_grade"`
	UseCount       int       `json:"use_count"`
	SuccessCount   int       `json:"success_count"`
	FailCount      int       `json:"fail_count"`
	LastUsed       time.Time `json:"last_used"`
	LastCheck      time.Time `json:"last_check"`
	CreatedAt      time.Time `json:"created_at"`
	Status         string    `json:"status"`
	Source         string    `json:"source"`          // "free" 或 "custom"
	SubscriptionID int64     `json:"subscription_id"` // 所属订阅ID（0=免费代理）
}

// Subscription 订阅信息
type Subscription struct {
	ID          int64     `json:"id"`
	Name        string    `json:"name"`
	GroupName   string    `json:"group_name"`
	URL         string    `json:"url"`
	FilePath    string    `json:"file_path"`
	Format      string    `json:"format"` // clash / plain / base64 / auto
	RefreshMin  int       `json:"refresh_min"`
	LastFetch   time.Time `json:"last_fetch"`
	LastSuccess time.Time `json:"last_success"` // 最后一次有可用节点的时间
	Status      string    `json:"status"`       // active / paused
	ProxyCount  int       `json:"proxy_count"`
	CreatedAt   time.Time `json:"created_at"`
	Contributed bool      `json:"contributed"` // 是否为访客贡献
}

// SourceStatus 代理源状态
type SourceStatus struct {
	ID               int64
	URL              string
	SuccessCount     int
	FailCount        int
	ConsecutiveFails int
	LastSuccess      time.Time
	LastFail         time.Time
	Status           string // active/degraded/disabled
	DisabledUntil    time.Time
}

type Storage struct {
	db                *sql.DB
	selectionSnapshot atomic.Pointer[proxySelectionSnapshot]
	snapshotTakenAt   atomic.Int64
	snapshotDirty     atomic.Bool
	snapshotMu        sync.Mutex
	pendingUseMu      sync.Mutex
	pendingSuccessUse map[string]int
	usageStopCh       chan struct{}
	usageWG           sync.WaitGroup
}

type proxySelectionSnapshot struct {
	all          []Proxy
	free         []Proxy
	custom       []Proxy
	httpOnly     []Proxy
	socks5Only   []Proxy
	freeHTTP     []Proxy
	freeSOCKS5   []Proxy
	customHTTP   []Proxy
	customSOCKS5 []Proxy
	byAddress    map[string]Proxy
}

const (
	successUseFlushInterval = 250 * time.Millisecond
	selectionSnapshotMinTTL = 2 * time.Second
)

func sqliteDSN(path string) string {
	if strings.Contains(path, "?") {
		return path + "&_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"
	}
	return path + "?_busy_timeout=5000&_journal_mode=WAL&_synchronous=NORMAL"
}

func New(dbPath string) (*Storage, error) {
	db, err := sql.Open("sqlite3", sqliteDSN(dbPath))
	if err != nil {
		return nil, fmt.Errorf("open db: %w", err)
	}

	// WAL 允许多读单写，保持小连接池即可显著减少高并发下的串行阻塞。
	db.SetMaxOpenConns(4)
	db.SetMaxIdleConns(4)

	s := &Storage{
		db:                db,
		pendingSuccessUse: make(map[string]int),
		usageStopCh:       make(chan struct{}),
	}
	if err := s.initSchema(); err != nil {
		return nil, err
	}
	s.startUsageFlusher()
	return s, nil
}

func (s *Storage) initSchema() error {
	// 创建代理表
	_, err := s.db.Exec(`
		CREATE TABLE IF NOT EXISTS proxies (
			id             INTEGER PRIMARY KEY AUTOINCREMENT,
			address        TEXT NOT NULL UNIQUE,
			protocol       TEXT NOT NULL,
			exit_ip        TEXT NOT NULL DEFAULT '',
			exit_location  TEXT NOT NULL DEFAULT '',
			latency        INTEGER NOT NULL DEFAULT 0,
			quality_grade  TEXT NOT NULL DEFAULT 'C',
			use_count      INTEGER NOT NULL DEFAULT 0,
			success_count  INTEGER NOT NULL DEFAULT 0,
			fail_count     INTEGER NOT NULL DEFAULT 0,
			last_used      DATETIME,
			last_check     DATETIME,
			created_at     DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
			status         TEXT NOT NULL DEFAULT 'active'
		)
	`)
	if err != nil {
		return err
	}

	// 创建索引
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_protocol_latency ON proxies(protocol, latency)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_quality_grade ON proxies(quality_grade, latency)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_status ON proxies(status)`)
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_select_active ON proxies(source, protocol, status, fail_count, latency)`)

	// 创建源状态表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS source_status (
			id                INTEGER PRIMARY KEY AUTOINCREMENT,
			url               TEXT NOT NULL UNIQUE,
			success_count     INTEGER NOT NULL DEFAULT 0,
			fail_count        INTEGER NOT NULL DEFAULT 0,
			consecutive_fails INTEGER NOT NULL DEFAULT 0,
			last_success      DATETIME,
			last_fail         DATETIME,
			status            TEXT NOT NULL DEFAULT 'active',
			disabled_until    DATETIME
		)
	`)
	if err != nil {
		return err
	}

	// 迁移：处理旧的 location 字段（如果存在）
	var hasOldLocation int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='location'`).Scan(&hasOldLocation)
	if err == nil && hasOldLocation > 0 {
		log.Println("[storage] migrating: renaming location to exit_location")
		// 如果有旧的 location 字段，先添加新字段再复制数据
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_location TEXT NOT NULL DEFAULT ''`)
		s.db.Exec(`UPDATE proxies SET exit_location = location WHERE location != ''`)
	}

	// 迁移：添加 exit_ip 字段
	var hasExitIP int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='exit_ip'`).Scan(&hasExitIP)
	if err == nil && hasExitIP == 0 {
		log.Println("[storage] migrating: adding exit_ip column")
		_, err = s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_ip TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("migrate exit_ip column: %w", err)
		}
	}

	// 迁移：添加 exit_location 字段
	var hasExitLocation int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='exit_location'`).Scan(&hasExitLocation)
	if err == nil && hasExitLocation == 0 {
		log.Println("[storage] migrating: adding exit_location column")
		_, err = s.db.Exec(`ALTER TABLE proxies ADD COLUMN exit_location TEXT NOT NULL DEFAULT ''`)
		if err != nil {
			return fmt.Errorf("migrate exit_location column: %w", err)
		}
	}

	// 迁移：添加 latency 字段
	var hasLatency int
	err = s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='latency'`).Scan(&hasLatency)
	if err == nil && hasLatency == 0 {
		log.Println("[storage] migrating: adding latency column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN latency INTEGER NOT NULL DEFAULT 0`)
	}

	// 迁移：添加质量等级字段
	var hasQuality int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='quality_grade'`).Scan(&hasQuality)
	if hasQuality == 0 {
		log.Println("[storage] migrating: adding quality_grade column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN quality_grade TEXT NOT NULL DEFAULT 'C'`)
	}

	// 迁移：添加使用统计字段
	var hasUseCount int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='use_count'`).Scan(&hasUseCount)
	if hasUseCount == 0 {
		log.Println("[storage] migrating: adding usage tracking columns")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN use_count INTEGER NOT NULL DEFAULT 0`)
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN success_count INTEGER NOT NULL DEFAULT 0`)
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN last_used DATETIME`)
	}

	// 迁移：添加状态字段
	var hasStatus int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='status'`).Scan(&hasStatus)
	if hasStatus == 0 {
		log.Println("[storage] migrating: adding status column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN status TEXT NOT NULL DEFAULT 'active'`)
	}

	// 迁移：添加 source 字段（区分免费代理和订阅代理）
	var hasSource int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='source'`).Scan(&hasSource)
	if hasSource == 0 {
		log.Println("[storage] migrating: adding source column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN source TEXT NOT NULL DEFAULT 'free'`)
	}
	s.db.Exec(`CREATE INDEX IF NOT EXISTS idx_source ON proxies(source, status)`)

	// 迁移：添加 subscription_id 字段
	var hasSubID int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('proxies') WHERE name='subscription_id'`).Scan(&hasSubID)
	if hasSubID == 0 {
		log.Println("[storage] migrating: adding subscription_id column")
		s.db.Exec(`ALTER TABLE proxies ADD COLUMN subscription_id INTEGER NOT NULL DEFAULT 0`)
	}

	// 创建订阅表
	_, err = s.db.Exec(`
		CREATE TABLE IF NOT EXISTS subscriptions (
			id            INTEGER PRIMARY KEY AUTOINCREMENT,
			name          TEXT NOT NULL DEFAULT '',
			group_name    TEXT NOT NULL DEFAULT '',
			url           TEXT NOT NULL DEFAULT '',
			file_path     TEXT NOT NULL DEFAULT '',
			format        TEXT NOT NULL DEFAULT 'clash',
			refresh_min   INTEGER NOT NULL DEFAULT 60,
			last_fetch    DATETIME,
			status        TEXT NOT NULL DEFAULT 'active',
			proxy_count   INTEGER NOT NULL DEFAULT 0,
			created_at    DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
		)
	`)
	if err != nil {
		return err
	}

	// 迁移：订阅表添加 contributed 和 last_success 字段
	var hasContributed int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='contributed'`).Scan(&hasContributed)
	if hasContributed == 0 {
		s.db.Exec(`ALTER TABLE subscriptions ADD COLUMN contributed INTEGER NOT NULL DEFAULT 0`)
	}
	var hasLastSuccess int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='last_success'`).Scan(&hasLastSuccess)
	if hasLastSuccess == 0 {
		s.db.Exec(`ALTER TABLE subscriptions ADD COLUMN last_success DATETIME`)
	}
	var hasGroupName int
	s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('subscriptions') WHERE name='group_name'`).Scan(&hasGroupName)
	if hasGroupName == 0 {
		s.db.Exec(`ALTER TABLE subscriptions ADD COLUMN group_name TEXT NOT NULL DEFAULT ''`)
	}

	return nil
}

// AddProxy 新增免费代理，已存在则忽略
func (s *Storage) AddProxy(address, protocol string) error {
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO proxies (address, protocol, source) VALUES (?, ?, 'free')`,
		address, protocol,
	)
	if err != nil {
		log.Printf("[storage] AddProxy %s error: %v", address, err)
		return err
	}

	// 检查是否真的插入了
	affected, _ := result.RowsAffected()
	if affected == 0 {
		log.Printf("[storage] AddProxy %s ignored (already exists or constraint)", address)
		return nil
	}
	s.invalidateSelectionSnapshot()
	return nil
}

// AddProxies 批量新增
func (s *Storage) AddProxies(proxies []Proxy) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	stmt, err := tx.Prepare(`INSERT OR IGNORE INTO proxies (address, protocol) VALUES (?, ?)`)
	if err != nil {
		tx.Rollback()
		return err
	}
	defer stmt.Close()

	for _, p := range proxies {
		if _, err := stmt.Exec(p.Address, p.Protocol); err != nil {
			log.Printf("insert proxy %s error: %v", p.Address, err)
		}
	}
	if err := tx.Commit(); err != nil {
		return err
	}
	if len(proxies) > 0 {
		s.invalidateSelectionSnapshot()
	}
	return nil
}

// proxyColumns 代理表查询的标准列列表
const proxyColumns = `id, address, protocol, exit_ip, exit_location, latency, quality_grade,
	use_count, success_count, fail_count, last_used, last_check, created_at, status, source, subscription_id`

// scanProxy 扫描代理行数据
func scanProxy(rows *sql.Rows) (*Proxy, error) {
	p := &Proxy{}
	var lastUsed, lastCheck sql.NullTime
	var source sql.NullString
	var subID sql.NullInt64
	if err := rows.Scan(&p.ID, &p.Address, &p.Protocol, &p.ExitIP, &p.ExitLocation,
		&p.Latency, &p.QualityGrade, &p.UseCount, &p.SuccessCount, &p.FailCount,
		&lastUsed, &lastCheck, &p.CreatedAt, &p.Status, &source, &subID); err != nil {
		return nil, err
	}
	if lastUsed.Valid {
		p.LastUsed = lastUsed.Time
	}
	if lastCheck.Valid {
		p.LastCheck = lastCheck.Time
	}
	if source.Valid {
		p.Source = source.String
	} else {
		p.Source = "free"
	}
	if subID.Valid {
		p.SubscriptionID = subID.Int64
	}
	return p, nil
}

func (s *Storage) invalidateSelectionSnapshot() {
	s.snapshotDirty.Store(false)
	s.snapshotTakenAt.Store(0)
	s.selectionSnapshot.Store(nil)
}

func (s *Storage) softInvalidateSnapshot() {
	snap := s.selectionSnapshot.Load()
	if snap == nil {
		return
	}
	taken := s.snapshotTakenAt.Load()
	if taken == 0 || time.Since(time.Unix(0, taken)) > selectionSnapshotMinTTL {
		s.invalidateSelectionSnapshot()
		return
	}
	s.snapshotDirty.Store(true)
}

func (s *Storage) snapshotExpiredAfterMinTTL() bool {
	if !s.snapshotDirty.Load() {
		return false
	}
	taken := s.snapshotTakenAt.Load()
	if taken == 0 {
		return true
	}
	return time.Since(time.Unix(0, taken)) > selectionSnapshotMinTTL
}

func (s *Storage) startUsageFlusher() {
	s.usageWG.Add(1)
	go func() {
		defer s.usageWG.Done()
		ticker := time.NewTicker(successUseFlushInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				_ = s.flushPendingSuccessUse()
			case <-s.usageStopCh:
				_ = s.flushPendingSuccessUse()
				return
			}
		}
	}()
}

func (s *Storage) flushPendingSuccessUse() error {
	s.pendingUseMu.Lock()
	if len(s.pendingSuccessUse) == 0 {
		s.pendingUseMu.Unlock()
		return nil
	}
	batch := s.pendingSuccessUse
	s.pendingSuccessUse = make(map[string]int, len(batch))
	s.pendingUseMu.Unlock()

	tx, err := s.db.Begin()
	if err != nil {
		s.restorePendingSuccessUse(batch)
		return err
	}

	stmt, err := tx.Prepare(
		`UPDATE proxies
		 SET use_count = use_count + ?, success_count = success_count + ?, fail_count = 0,
		     last_used = CURRENT_TIMESTAMP
		 WHERE address = ?`,
	)
	if err != nil {
		_ = tx.Rollback()
		s.restorePendingSuccessUse(batch)
		return err
	}
	defer stmt.Close()

	for address, count := range batch {
		if _, err := stmt.Exec(count, count, address); err != nil {
			_ = tx.Rollback()
			s.restorePendingSuccessUse(batch)
			return err
		}
	}

	if err := tx.Commit(); err != nil {
		s.restorePendingSuccessUse(batch)
		return err
	}
	return nil
}

func (s *Storage) restorePendingSuccessUse(batch map[string]int) {
	if len(batch) == 0 {
		return
	}
	s.pendingUseMu.Lock()
	defer s.pendingUseMu.Unlock()

	if s.pendingSuccessUse == nil {
		s.pendingSuccessUse = make(map[string]int, len(batch))
	}
	for address, count := range batch {
		s.pendingSuccessUse[address] += count
	}
}

func (s *Storage) getSelectionSnapshot() (*proxySelectionSnapshot, error) {
	if snap := s.selectionSnapshot.Load(); snap != nil && !s.snapshotExpiredAfterMinTTL() {
		return snap, nil
	}

	s.snapshotMu.Lock()
	defer s.snapshotMu.Unlock()

	if snap := s.selectionSnapshot.Load(); snap != nil && !s.snapshotExpiredAfterMinTTL() {
		return snap, nil
	}

	rows, err := s.db.Query(
		`SELECT ` + proxyColumns + `
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3
		 ORDER BY latency ASC, id ASC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	snap := buildSelectionSnapshot(proxies)
	s.snapshotDirty.Store(false)
	s.snapshotTakenAt.Store(time.Now().UnixNano())
	s.selectionSnapshot.Store(snap)
	return snap, nil
}

func buildSelectionSnapshot(proxies []Proxy) *proxySelectionSnapshot {
	snap := &proxySelectionSnapshot{
		all:          make([]Proxy, 0, len(proxies)),
		free:         make([]Proxy, 0, len(proxies)),
		custom:       make([]Proxy, 0, len(proxies)),
		httpOnly:     make([]Proxy, 0, len(proxies)),
		socks5Only:   make([]Proxy, 0, len(proxies)),
		freeHTTP:     make([]Proxy, 0, len(proxies)),
		freeSOCKS5:   make([]Proxy, 0, len(proxies)),
		customHTTP:   make([]Proxy, 0, len(proxies)),
		customSOCKS5: make([]Proxy, 0, len(proxies)),
		byAddress:    make(map[string]Proxy, len(proxies)),
	}

	for _, p := range proxies {
		snap.all = append(snap.all, p)
		snap.byAddress[p.Address] = p

		switch p.Source {
		case "custom":
			snap.custom = append(snap.custom, p)
			if p.Protocol == "http" {
				snap.customHTTP = append(snap.customHTTP, p)
			} else if p.Protocol == "socks5" {
				snap.customSOCKS5 = append(snap.customSOCKS5, p)
			}
		default:
			snap.free = append(snap.free, p)
			if p.Protocol == "http" {
				snap.freeHTTP = append(snap.freeHTTP, p)
			} else if p.Protocol == "socks5" {
				snap.freeSOCKS5 = append(snap.freeSOCKS5, p)
			}
		}

		if p.Protocol == "http" {
			snap.httpOnly = append(snap.httpOnly, p)
		} else if p.Protocol == "socks5" {
			snap.socks5Only = append(snap.socks5Only, p)
		}
	}

	return snap
}

func (s *proxySelectionSnapshot) selectList(sourceFilter, protocol string) []Proxy {
	switch sourceFilter {
	case "free":
		switch protocol {
		case "http":
			return s.freeHTTP
		case "socks5":
			return s.freeSOCKS5
		default:
			return s.free
		}
	case "custom":
		switch protocol {
		case "http":
			return s.customHTTP
		case "socks5":
			return s.customSOCKS5
		default:
			return s.custom
		}
	default:
		switch protocol {
		case "http":
			return s.httpOnly
		case "socks5":
			return s.socks5Only
		default:
			return s.all
		}
	}
}

func buildExcludeSet(excludes []string) map[string]struct{} {
	if len(excludes) == 0 {
		return nil
	}
	set := make(map[string]struct{}, len(excludes))
	for _, address := range excludes {
		set[address] = struct{}{}
	}
	return set
}

func matchesSnapshotFilters(p Proxy, region, state string, excludes map[string]struct{}) bool {
	if len(excludes) > 0 {
		if _, found := excludes[p.Address]; found {
			return false
		}
	}

	if region != "" {
		fields := strings.Fields(strings.ToUpper(p.ExitLocation))
		if len(fields) == 0 || fields[0] != region {
			return false
		}
	}

	if state != "" {
		location := strings.ToUpper(strings.TrimSpace(p.ExitLocation))
		if location == "" || !strings.Contains(location, state) {
			return false
		}
	}

	return true
}

// GetAll 获取所有可用代理
func (s *Storage) GetAll() ([]Proxy, error) {
	return s.GetAllFiltered("")
}

// GetAllFiltered 获取可用代理（可按来源过滤）
// sourceFilter: "" = 全部, "free" = 仅免费, "custom" = 仅订阅
func (s *Storage) GetAllFiltered(sourceFilter string) ([]Proxy, error) {
	query := `SELECT ` + proxyColumns + `
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3`
	var args []interface{}
	if sourceFilter != "" {
		query += ` AND source = ?`
		args = append(args, sourceFilter)
	}
	query += ` ORDER BY latency ASC`

	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// GetByAddress 按地址获取可用代理
func (s *Storage) GetByAddress(address string) (*Proxy, error) {
	snap, err := s.getSelectionSnapshot()
	if err != nil {
		return nil, err
	}
	if proxy, ok := snap.byAddress[address]; ok {
		result := proxy
		return &result, nil
	}
	return nil, fmt.Errorf("proxy not found: %s", address)
}

// SelectProxy 按过滤条件直接从数据库中选出一个代理，避免热路径全表加载。
func (s *Storage) SelectProxy(
	sourceFilter string,
	protocol string,
	region string,
	state string,
	excludes []string,
	lowestLatency bool,
) (*Proxy, error) {
	snap, err := s.getSelectionSnapshot()
	if err != nil {
		return nil, err
	}

	candidates := snap.selectList(sourceFilter, protocol)
	excludeSet := buildExcludeSet(excludes)

	if lowestLatency {
		for i := range candidates {
			if !matchesSnapshotFilters(candidates[i], region, state, excludeSet) {
				continue
			}
			proxy := candidates[i]
			return &proxy, nil
		}
		return nil, fmt.Errorf("no available proxy")
	}

	var selected *Proxy
	seen := 0
	for i := range candidates {
		if !matchesSnapshotFilters(candidates[i], region, state, excludeSet) {
			continue
		}
		seen++
		if rand.Intn(seen) == 0 {
			proxy := candidates[i]
			selected = &proxy
		}
	}
	if selected != nil {
		return selected, nil
	}
	return nil, fmt.Errorf("no available proxy")
}

// Delete 立即删除指定代理
func (s *Storage) Delete(address string) error {
	result, err := s.db.Exec(`DELETE FROM proxies WHERE address = ?`, address)
	if err != nil {
		return err
	}
	if affected, _ := result.RowsAffected(); affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// IncrFail 增加失败次数
func (s *Storage) IncrFail(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = fail_count + 1, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// ResetFail 重置失败次数（验证通过）
func (s *Storage) ResetFail(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = 0, last_check = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// UpdateLatency 更新代理的延迟信息（毫秒）
func (s *Storage) UpdateLatency(address string, latencyMs int) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET latency = ? WHERE address = ?`,
		latencyMs, address,
	)
	if err == nil {
		s.softInvalidateSnapshot()
	}
	return err
}

// UpdateExitInfo 更新代理的出口 IP、位置和质量等级
func (s *Storage) UpdateExitInfo(address, exitIP, exitLocation string, latencyMs int) error {
	grade := CalculateQualityGrade(latencyMs)
	_, err := s.db.Exec(
		`UPDATE proxies
		 SET exit_ip = ?, exit_location = ?, latency = ?, quality_grade = ?, fail_count = 0, last_check = CURRENT_TIMESTAMP
		 WHERE address = ?`,
		exitIP, exitLocation, latencyMs, grade, address,
	)
	if err == nil {
		s.softInvalidateSnapshot()
	}
	return err
}

// RecordProxyUse 记录代理使用（成功）
func (s *Storage) RecordProxyUse(address string, success bool) error {
	if success {
		s.pendingUseMu.Lock()
		if s.pendingSuccessUse != nil {
			s.pendingSuccessUse[address]++
		}
		s.pendingUseMu.Unlock()
		return nil
	}
	_, err := s.db.Exec(
		`UPDATE proxies SET use_count = use_count + 1, fail_count = fail_count + 1, 
		 last_used = CURRENT_TIMESTAMP WHERE address = ?`,
		address,
	)
	return err
}

// GetWorstProxies 获取指定协议中延迟最高的N个代理（仅免费代理）
func (s *Storage) GetWorstProxies(protocol string, limit int) ([]Proxy, error) {
	rows, err := s.db.Query(
		`SELECT `+proxyColumns+`
		 FROM proxies
		 WHERE protocol = ? AND status = 'active' AND source = 'free'
		   AND quality_grade != 'S'
		   AND (JULIANDAY('now') - JULIANDAY(created_at)) * 1440 > 60
		 ORDER BY latency DESC, fail_count DESC
		 LIMIT ?`, protocol, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// ReplaceProxy 替换代理（删除旧的，添加新的）
func (s *Storage) ReplaceProxy(oldAddress string, newProxy Proxy) error {
	tx, err := s.db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	// 删除旧代理
	_, err = tx.Exec(`DELETE FROM proxies WHERE address = ?`, oldAddress)
	if err != nil {
		return err
	}

	// 添加新代理（带完整信息）
	grade := CalculateQualityGrade(newProxy.Latency)
	source := newProxy.Source
	if source == "" {
		source = "free"
	}
	_, err = tx.Exec(
		`INSERT INTO proxies (address, protocol, exit_ip, exit_location, latency, quality_grade, status, source)
		 VALUES (?, ?, ?, ?, ?, ?, 'active', ?)`,
		newProxy.Address, newProxy.Protocol, newProxy.ExitIP, newProxy.ExitLocation, newProxy.Latency, grade, source,
	)
	if err != nil {
		return err
	}

	if err := tx.Commit(); err != nil {
		return err
	}
	s.invalidateSelectionSnapshot()
	return nil
}

// MarkAsReplacementCandidate 标记代理为替换候选
func (s *Storage) MarkAsReplacementCandidate(addresses []string) error {
	if len(addresses) == 0 {
		return nil
	}
	placeholders := make([]string, len(addresses))
	args := make([]interface{}, len(addresses))
	for i, addr := range addresses {
		placeholders[i] = "?"
		args[i] = addr
	}
	query := fmt.Sprintf(`UPDATE proxies SET status = 'candidate_replace' WHERE address IN (%s)`,
		strings.Join(placeholders, ","))
	_, err := s.db.Exec(query, args...)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// GetAverageLatency 获取指定协议的平均延迟
func (s *Storage) GetAverageLatency(protocol string) (int, error) {
	var avg sql.NullFloat64
	err := s.db.QueryRow(
		`SELECT AVG(latency) FROM proxies WHERE protocol = ? AND status = 'active' AND latency > 0`,
		protocol,
	).Scan(&avg)
	if err != nil || !avg.Valid {
		return 0, err
	}
	return int(avg.Float64), nil
}

// GetQualityDistribution 获取质量分布统计
func (s *Storage) GetQualityDistribution() (map[string]int, error) {
	rows, err := s.db.Query(
		`SELECT quality_grade, COUNT(*) as count 
		 FROM proxies 
		 WHERE status = 'active' AND fail_count < 3
		 GROUP BY quality_grade`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	dist := make(map[string]int)
	for rows.Next() {
		var grade string
		var count int
		if err := rows.Scan(&grade, &count); err != nil {
			return nil, err
		}
		dist[grade] = count
	}
	return dist, nil
}

// GetBatchForHealthCheck 获取一批需要健康检查的代理。
// sourceFilter 为空时检查全部来源；非空时仅检查指定来源。
func (s *Storage) GetBatchForHealthCheck(batchSize int, skipSGrade bool, sourceFilter string) ([]Proxy, error) {
	query := `SELECT ` + proxyColumns + `
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3`
	args := []interface{}{}

	if sourceFilter != "" {
		query += ` AND source = ?`
		args = append(args, sourceFilter)
	}

	if skipSGrade {
		query += ` AND quality_grade != 'S'`
	}

	query += ` ORDER BY 
		COALESCE(last_check, '1970-01-01') ASC,
		quality_grade DESC
		LIMIT ?`

	args = append(args, batchSize)
	rows, err := s.db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// CalculateQualityGrade 根据当前配置阈值计算质量等级。
// S/A 是健康阈值内的细分，B 代表仍在标准可用范围内，C 代表超过标准阈值。
func CalculateQualityGrade(latencyMs int) string {
	return CalculateQualityGradeForConfig(latencyMs, config.Get())
}

// CalculateQualityGradeForConfig 根据指定配置阈值计算质量等级。
func CalculateQualityGradeForConfig(latencyMs int, cfg *config.Config) string {
	healthyThreshold := 2000
	maxThreshold := 2500
	if cfg != nil {
		if cfg.MaxLatencyHealthy > 0 {
			healthyThreshold = cfg.MaxLatencyHealthy
		}
		if cfg.MaxLatencyMs > 0 {
			maxThreshold = cfg.MaxLatencyMs
		}
	}
	if healthyThreshold < 1 {
		healthyThreshold = 1
	}
	if maxThreshold < healthyThreshold {
		maxThreshold = healthyThreshold
	}

	sThreshold := healthyThreshold / 4
	if sThreshold < 1 {
		sThreshold = 1
	}
	aThreshold := healthyThreshold / 2
	if aThreshold < sThreshold {
		aThreshold = sThreshold
	}

	switch {
	case latencyMs <= sThreshold:
		return "S" // 超快
	case latencyMs <= aThreshold:
		return "A" // 良好
	case latencyMs <= maxThreshold:
		return "B" // 可用
	default:
		return "C" // 慢速
	}
}

// DeleteInvalid 删除失败次数超过阈值的代理（仅免费代理）
func (s *Storage) DeleteInvalid(maxFailCount int) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE fail_count >= ? AND source = 'free'`, maxFailCount)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// DeleteBlockedCountries 删除指定国家代码出口的代理
func (s *Storage) DeleteBlockedCountries(countryCodes []string) (int64, error) {
	if len(countryCodes) == 0 {
		return 0, nil
	}

	var totalDeleted int64
	for _, code := range countryCodes {
		// exit_location 格式：如 "CN Beijing" 或 "CN"（仅国家代码）
		// 同时匹配 "CODE" 和 "CODE ..." 两种情况（仅删除免费代理）
		res, err := s.db.Exec(`DELETE FROM proxies WHERE source = 'free' AND (exit_location = ? OR exit_location LIKE ?)`, code, code+" %")
		if err != nil {
			return totalDeleted, err
		}
		affected, _ := res.RowsAffected()
		totalDeleted += affected
	}
	if totalDeleted > 0 {
		s.invalidateSelectionSnapshot()
	}
	return totalDeleted, nil
}

// DeleteNotAllowedCountries 删除不在白名单中的代理
func (s *Storage) DeleteNotAllowedCountries(allowedCodes []string) (int64, error) {
	if len(allowedCodes) == 0 {
		return 0, nil
	}

	// 构建 WHERE 条件：exit_location 不以任何白名单国家代码开头
	// 即：NOT (exit_location = 'US' OR exit_location LIKE 'US %' OR ...)
	conditions := make([]string, 0, len(allowedCodes)*2)
	args := make([]interface{}, 0, len(allowedCodes)*2)
	for _, code := range allowedCodes {
		conditions = append(conditions, "exit_location = ?", "exit_location LIKE ?")
		args = append(args, code, code+" %")
	}

	query := `DELETE FROM proxies WHERE source = 'free' AND exit_location != '' AND NOT (` + strings.Join(conditions, " OR ") + `)`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// DeleteWithoutExitInfo 删除没有出口信息的代理（仅免费代理）
func (s *Storage) DeleteWithoutExitInfo() (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE source = 'free' AND (exit_ip = '' OR exit_location = '')`)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// DisableBlockedCountries 禁用订阅代理中属于被屏蔽国家的（不删除）
func (s *Storage) DisableBlockedCountries(countryCodes []string) (int64, error) {
	if len(countryCodes) == 0 {
		return 0, nil
	}
	var total int64
	for _, code := range countryCodes {
		res, err := s.db.Exec(
			`UPDATE proxies SET status = 'disabled' WHERE source = 'custom' AND status = 'active' AND (exit_location = ? OR exit_location LIKE ?)`,
			code, code+" %",
		)
		if err != nil {
			return total, err
		}
		affected, _ := res.RowsAffected()
		total += affected
	}
	if total > 0 {
		s.invalidateSelectionSnapshot()
	}
	return total, nil
}

// DisableNotAllowedCountries 禁用订阅代理中不在白名单的（不删除）
func (s *Storage) DisableNotAllowedCountries(allowedCodes []string) (int64, error) {
	if len(allowedCodes) == 0 {
		return 0, nil
	}
	conditions := make([]string, 0, len(allowedCodes)*2)
	args := make([]interface{}, 0, len(allowedCodes)*2)
	for _, code := range allowedCodes {
		conditions = append(conditions, "exit_location = ?", "exit_location LIKE ?")
		args = append(args, code, code+" %")
	}
	query := `UPDATE proxies SET status = 'disabled' WHERE source = 'custom' AND status = 'active' AND exit_location != '' AND NOT (` + strings.Join(conditions, " OR ") + `)`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// Count 返回可用代理数量（仅免费代理，用于 slot 计算）
func (s *Storage) Count() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3 AND source = 'free'`,
	).Scan(&count)
	return count, err
}

// CountAll 返回所有可用代理数量（免费+订阅）
func (s *Storage) CountAll() (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3`,
	).Scan(&count)
	return count, err
}

// CountByProtocol 按协议统计数量（仅免费代理，用于 slot 计算）
func (s *Storage) CountByProtocol(protocol string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3 AND source = 'free' AND protocol = ?`,
		protocol,
	).Scan(&count)
	return count, err
}

// CountByProtocolAll 按协议统计所有可用代理数量（免费+订阅）。
func (s *Storage) CountByProtocolAll(protocol string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE status IN ('active', 'degraded') AND fail_count < 3 AND protocol = ?`,
		protocol,
	).Scan(&count)
	return count, err
}

// IncrementFailCount 增加失败次数
func (s *Storage) IncrementFailCount(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET fail_count = fail_count + 1 WHERE address = ?`,
		address,
	)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// GetByProtocol 按协议获取代理列表
func (s *Storage) GetByProtocol(protocol string) ([]Proxy, error) {
	rows, err := s.db.Query(
		`SELECT `+proxyColumns+`
		 FROM proxies
		 WHERE status IN ('active', 'degraded') AND fail_count < 3 AND protocol = ?
		 ORDER BY latency ASC`, protocol,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// ========== 订阅代理相关方法 ==========

// AddProxyWithSource 新增代理并指定来源和订阅ID
func (s *Storage) AddProxyWithSource(address, protocol, source string, subscriptionID ...int64) error {
	subID := int64(0)
	if len(subscriptionID) > 0 {
		subID = subscriptionID[0]
	}
	status := "active"
	if source == "custom" {
		status = "disabled"
	}
	result, err := s.db.Exec(
		`INSERT OR IGNORE INTO proxies (address, protocol, source, subscription_id, status, fail_count)
		 VALUES (?, ?, ?, ?, ?, 0)`,
		address, protocol, source, subID, status,
	)
	if err != nil {
		log.Printf("[storage] AddProxyWithSource %s error: %v", address, err)
		return err
	}
	affected, _ := result.RowsAffected()
	if affected == 0 {
		// 已存在，更新 source、订阅归属和初始状态。
		_, err = s.db.Exec(
			`UPDATE proxies SET source = ?, subscription_id = ?, status = ?, fail_count = 0 WHERE address = ?`,
			source, subID, status, address,
		)
	}
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// DeleteBySubscriptionID 删除指定订阅的所有代理
func (s *Storage) DeleteBySubscriptionID(subscriptionID int64) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE subscription_id = ?`, subscriptionID)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// DisableProxy 禁用代理（软删除，用于订阅代理）
func (s *Storage) DisableProxy(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET status = 'disabled' WHERE address = ?`,
		address,
	)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// EnableProxy 启用代理（从禁用状态恢复）
func (s *Storage) EnableProxy(address string) error {
	_, err := s.db.Exec(
		`UPDATE proxies SET status = 'active', fail_count = 0 WHERE address = ?`,
		address,
	)
	if err == nil {
		s.invalidateSelectionSnapshot()
	}
	return err
}

// GetDisabledCustomProxies 获取所有被禁用的订阅代理
func (s *Storage) GetDisabledCustomProxies() ([]Proxy, error) {
	rows, err := s.db.Query(
		`SELECT ` + proxyColumns + `
		 FROM proxies
		 WHERE source = 'custom' AND status = 'disabled'`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var proxies []Proxy
	for rows.Next() {
		p, err := scanProxy(rows)
		if err != nil {
			return nil, err
		}
		proxies = append(proxies, *p)
	}
	return proxies, nil
}

// CountBySource 按来源统计可用代理数量
func (s *Storage) CountBySource(source string) (int, error) {
	var count int
	err := s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE source = ? AND status IN ('active', 'degraded') AND fail_count < 3`,
		source,
	).Scan(&count)
	return count, err
}

// DeleteBySource 删除指定来源的所有代理
func (s *Storage) DeleteBySource(source string) (int64, error) {
	res, err := s.db.Exec(`DELETE FROM proxies WHERE source = ?`, source)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// DeleteCustomProxiesNotIn 删除不在给定地址列表中的订阅代理
func (s *Storage) DeleteCustomProxiesNotIn(addresses []string) (int64, error) {
	if len(addresses) == 0 {
		return s.DeleteBySource("custom")
	}
	placeholders := make([]string, len(addresses))
	args := make([]interface{}, len(addresses))
	for i, addr := range addresses {
		placeholders[i] = "?"
		args[i] = addr
	}
	query := `DELETE FROM proxies WHERE source = 'custom' AND address NOT IN (` + strings.Join(placeholders, ",") + `)`
	res, err := s.db.Exec(query, args...)
	if err != nil {
		return 0, err
	}
	affected, err := res.RowsAffected()
	if affected > 0 {
		s.invalidateSelectionSnapshot()
	}
	return affected, err
}

// ========== 订阅 CRUD ==========

// AddSubscription 添加订阅（自动去重：相同 URL 或 file_path 不重复添加）
func (s *Storage) AddSubscription(name, groupName, url, filePath, format string, refreshMin int) (int64, error) {
	// 去重检查
	if url != "" {
		var existID int64
		err := s.db.QueryRow(`SELECT id FROM subscriptions WHERE url = ? AND url != ''`, url).Scan(&existID)
		if err == nil {
			return 0, fmt.Errorf("该订阅 URL 已存在")
		}
	}
	if filePath != "" {
		var existID int64
		err := s.db.QueryRow(`SELECT id FROM subscriptions WHERE file_path = ? AND file_path != ''`, filePath).Scan(&existID)
		if err == nil {
			return 0, fmt.Errorf("该订阅文件已存在")
		}
	}

	res, err := s.db.Exec(
		`INSERT INTO subscriptions (name, group_name, url, file_path, format, refresh_min) VALUES (?, ?, ?, ?, ?, ?)`,
		name, strings.TrimSpace(groupName), url, filePath, format, refreshMin,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// CountBySubscriptionID 统计指定订阅的可用/禁用代理数
func (s *Storage) CountBySubscriptionID(subID int64) (active int, disabled int) {
	s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE subscription_id = ? AND status IN ('active', 'degraded') AND fail_count < 3`,
		subID,
	).Scan(&active)
	s.db.QueryRow(
		`SELECT COUNT(*) FROM proxies WHERE subscription_id = ? AND status = 'disabled'`,
		subID,
	).Scan(&disabled)
	return
}

// AddContributedSubscription 添加访客贡献的订阅
func (s *Storage) AddContributedSubscription(name, url string, refreshMin int) (int64, error) {
	if url == "" {
		return 0, fmt.Errorf("URL 不能为空")
	}
	// 去重
	var existID int64
	err := s.db.QueryRow(`SELECT id FROM subscriptions WHERE url = ? AND url != ''`, url).Scan(&existID)
	if err == nil {
		return 0, fmt.Errorf("该订阅 URL 已存在")
	}

	res, err := s.db.Exec(
		`INSERT INTO subscriptions (name, group_name, url, format, refresh_min, contributed) VALUES (?, '', ?, 'auto', ?, 1)`,
		name, url, refreshMin,
	)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

// UpdateSubscription 更新订阅
func (s *Storage) UpdateSubscription(id int64, name, groupName, url, filePath, format string, refreshMin int) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions SET name = ?, group_name = ?, url = ?, file_path = ?, format = ?, refresh_min = ? WHERE id = ?`,
		name, strings.TrimSpace(groupName), url, filePath, format, refreshMin, id,
	)
	return err
}

func (s *Storage) SetSubscriptionGroup(id int64, groupName string) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions SET group_name = ? WHERE id = ?`,
		strings.TrimSpace(groupName), id,
	)
	return err
}

// DeleteSubscription 删除订阅
func (s *Storage) DeleteSubscription(id int64) error {
	_, err := s.db.Exec(`DELETE FROM subscriptions WHERE id = ?`, id)
	return err
}

// GetSubscriptions 获取所有订阅
func (s *Storage) GetSubscriptions() ([]Subscription, error) {
	rows, err := s.db.Query(
		`SELECT ` + subColumns + `
		 FROM subscriptions ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *sub)
	}
	return subs, nil
}

// GetSubscription 获取单个订阅
func (s *Storage) GetSubscription(id int64) (*Subscription, error) {
	rows, err := s.db.Query(
		`SELECT `+subColumns+`
		 FROM subscriptions WHERE id = ?`, id,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if rows.Next() {
		return scanSubscription(rows)
	}
	return nil, fmt.Errorf("subscription %d not found", id)
}

// UpdateSubscriptionFetch 更新订阅的最后拉取时间和代理数
func (s *Storage) UpdateSubscriptionFetch(id int64, proxyCount int) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions SET last_fetch = CURRENT_TIMESTAMP, proxy_count = ? WHERE id = ?`,
		proxyCount, id,
	)
	return err
}

// UpdateSubscriptionSuccess 记录订阅最后一次有可用节点的时间
func (s *Storage) UpdateSubscriptionSuccess(id int64) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions SET last_success = CURRENT_TIMESTAMP WHERE id = ?`, id,
	)
	return err
}

// GetStaleSubscriptions 获取连续 N 天无可用节点的订阅
func (s *Storage) GetStaleSubscriptions(staleDays int) ([]Subscription, error) {
	rows, err := s.db.Query(
		`SELECT `+subColumns+`
		 FROM subscriptions
		 WHERE status = 'active'
		   AND (last_success IS NULL OR JULIANDAY('now') - JULIANDAY(last_success) > ?)
		   AND JULIANDAY('now') - JULIANDAY(created_at) > ?`,
		staleDays, staleDays,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var subs []Subscription
	for rows.Next() {
		sub, err := scanSubscription(rows)
		if err != nil {
			return nil, err
		}
		subs = append(subs, *sub)
	}
	return subs, nil
}

// ToggleSubscription 切换订阅状态
func (s *Storage) ToggleSubscription(id int64) error {
	_, err := s.db.Exec(
		`UPDATE subscriptions SET status = CASE WHEN status = 'active' THEN 'paused' ELSE 'active' END WHERE id = ?`,
		id,
	)
	return err
}

// scanSubscription 扫描订阅行数据
// subColumns 订阅表查询列
const subColumns = `id, name, group_name, url, file_path, format, refresh_min, last_fetch, last_success, status, proxy_count, created_at, contributed`

func scanSubscription(rows *sql.Rows) (*Subscription, error) {
	sub := &Subscription{}
	var lastFetch, lastSuccess sql.NullTime
	var contributed int
	if err := rows.Scan(&sub.ID, &sub.Name, &sub.GroupName, &sub.URL, &sub.FilePath, &sub.Format,
		&sub.RefreshMin, &lastFetch, &lastSuccess, &sub.Status, &sub.ProxyCount, &sub.CreatedAt, &contributed); err != nil {
		return nil, err
	}
	if lastFetch.Valid {
		sub.LastFetch = lastFetch.Time
	}
	if lastSuccess.Valid {
		sub.LastSuccess = lastSuccess.Time
	}
	sub.Contributed = contributed == 1
	return sub, nil
}

// Close 关闭数据库
func (s *Storage) Close() error {
	if s.usageStopCh != nil {
		close(s.usageStopCh)
		s.usageWG.Wait()
		s.usageStopCh = nil
	}
	return s.db.Close()
}

// GetDB 获取数据库实例（供其他模块使用）
func (s *Storage) GetDB() *sql.DB {
	return s.db
}
