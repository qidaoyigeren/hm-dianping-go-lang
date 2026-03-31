package utils

import (
	"context"
	"fmt"
	"log"
	"strconv"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// BloomFilterConfig 布隆过滤器配置
type BloomFilterConfig struct {
	Key        string  // 过滤器键名
	ErrorRate  float64 // 误判率
	Capacity   uint    // 初始容量
	Expansion  uint    // 扩容倍数，默认2
	NonScaling bool    // 是否禁用自动扩容
}

// BloomFilter 布隆过滤器操作接口
type BloomFilter struct {
	config BloomFilterConfig
	rdb    *redis.Client
	mu     sync.RWMutex // 保护配置的并发安全
}

// BloomFilterStats 布隆过滤器统计信息
type BloomFilterStats struct {
	TotalQueries  int64     // 总查询次数
	HitCount      int64     // 命中次数（可能存在）
	MissCount     int64     // 未命中次数（肯定不存在）
	ErrorCount    int64     // 错误次数
	LastErrorTime time.Time // 最后一次错误时间
	IsDegraded    bool      // 是否处于降级模式
}

// BloomFilterWithFallback 带降级机制的布隆过滤器包装器
type BloomFilterWithFallback struct {
	bf      *BloomFilter
	stats   *BloomFilterStats
	mu      sync.RWMutex
	onError func(error) // 错误回调
}

// NewBloomFilter 创建新的布隆过滤器实例
func NewBloomFilter(rdb *redis.Client, config BloomFilterConfig) *BloomFilter {
	// 设置默认值
	if config.ErrorRate == 0 {
		config.ErrorRate = 0.01 // 默认1%误判率
	}
	if config.Capacity == 0 {
		config.Capacity = 10000 // 默认1万容量
	}
	if config.Expansion == 0 {
		config.Expansion = 2 // 默认2倍扩容
	}

	return &BloomFilter{
		config: config,
		rdb:    rdb,
	}
}

// Reserve 创建布隆过滤器
func (bf *BloomFilter) Reserve(ctx context.Context) error {
	args := []interface{}{"BF.RESERVE", bf.config.Key, bf.config.ErrorRate, bf.config.Capacity}

	// 添加可选参数
	if bf.config.Expansion != 2 {
		args = append(args, "EXPANSION", bf.config.Expansion)
	}
	if bf.config.NonScaling {
		args = append(args, "NONSCALING")
	}

	err := bf.rdb.Do(ctx, args...).Err()
	if err != nil {
		// 如果过滤器已存在，忽略错误
		if err.Error() == "ERR item exists" {
			return nil
		}
		return fmt.Errorf("创建布隆过滤器失败: %v", err)
	}
	return nil
}

// Add 添加单个元素到布隆过滤器
func (bf *BloomFilter) Add(ctx context.Context, item string) (bool, error) {
	cmd := bf.rdb.Do(ctx, "BF.ADD", bf.config.Key, item)
	if err := cmd.Err(); err != nil {
		return false, fmt.Errorf("添加元素到布隆过滤器失败: %v", err)
	}

	result, err := cmd.Result()
	if err != nil {
		return false, fmt.Errorf("解析布隆过滤器结果失败: %v", err)
	}

	// 处理不同的返回类型：BF.ADD 返回 1（新元素）或 0（已存在）
	switch v := result.(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("未知的返回类型: %T", result)
	}
}

// AddMulti 批量添加元素到布隆过滤器
func (bf *BloomFilter) AddMulti(ctx context.Context, items []string) ([]bool, error) {
	if len(items) == 0 {
		return []bool{}, nil
	}

	args := []interface{}{"BF.MADD", bf.config.Key}
	for _, item := range items {
		args = append(args, item)
	}

	results, err := bf.rdb.Do(ctx, args...).Slice()
	if err != nil {
		return nil, fmt.Errorf("批量添加元素到布隆过滤器失败: %v", err)
	}

	boolResults := make([]bool, len(results))
	for i, result := range results {
		if val, ok := result.(int64); ok {
			boolResults[i] = val == 1
		}
	}

	return boolResults, nil
}

// Exists 检查单个元素是否存在于布隆过滤器中
func (bf *BloomFilter) Exists(ctx context.Context, item string) (bool, error) {
	cmd := bf.rdb.Do(ctx, "BF.EXISTS", bf.config.Key, item)
	if err := cmd.Err(); err != nil {
		return false, fmt.Errorf("检查布隆过滤器元素失败: %v", err)
	}

	result, err := cmd.Result()
	if err != nil {
		return false, fmt.Errorf("解析布隆过滤器结果失败: %v", err)
	}

	// 处理不同的返回类型
	switch v := result.(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	case bool:
		return v, nil
	default:
		return false, fmt.Errorf("未知的返回类型: %T", result)
	}
}

// ExistsMulti 批量检查元素是否存在于布隆过滤器中
func (bf *BloomFilter) ExistsMulti(ctx context.Context, items []string) ([]bool, error) {
	if len(items) == 0 {
		return []bool{}, nil
	}

	args := []interface{}{"BF.MEXISTS", bf.config.Key}
	for _, item := range items {
		args = append(args, item)
	}

	results, err := bf.rdb.Do(ctx, args...).Slice()
	if err != nil {
		return nil, fmt.Errorf("批量检查布隆过滤器元素失败: %v", err)
	}

	boolResults := make([]bool, len(results))
	for i, result := range results {
		if val, ok := result.(int64); ok {
			boolResults[i] = val == 1
		}
	}

	return boolResults, nil
}

// Info 获取布隆过滤器信息
func (bf *BloomFilter) Info(ctx context.Context) (map[string]interface{}, error) {
	results, err := bf.rdb.Do(ctx, "BF.INFO", bf.config.Key).Slice()
	if err != nil {
		return nil, fmt.Errorf("获取布隆过滤器信息失败: %v", err)
	}

	info := make(map[string]interface{})
	for i := 0; i < len(results); i += 2 {
		if i+1 < len(results) {
			key := fmt.Sprintf("%v", results[i])
			value := results[i+1]
			info[key] = value
		}
	}

	return info, nil
}

// Delete 删除布隆过滤器
func (bf *BloomFilter) Delete(ctx context.Context) error {
	err := bf.rdb.Del(ctx, bf.config.Key).Err()
	if err != nil {
		return fmt.Errorf("删除布隆过滤器失败: %v", err)
	}
	return nil
}

// 便利函数：将数字ID转换为字符串

// AddID 添加数字ID到布隆过滤器
func (bf *BloomFilter) AddID(ctx context.Context, id uint) (bool, error) {
	return bf.Add(ctx, strconv.FormatUint(uint64(id), 10))
}

// ExistsID 检查数字ID是否存在于布隆过滤器中
func (bf *BloomFilter) ExistsID(ctx context.Context, id uint) (bool, error) {
	return bf.Exists(ctx, strconv.FormatUint(uint64(id), 10))
}

// AddIDs 批量添加数字ID到布隆过滤器
func (bf *BloomFilter) AddIDs(ctx context.Context, ids []uint) ([]bool, error) {
	items := make([]string, len(ids))
	for i, id := range ids {
		items[i] = strconv.FormatUint(uint64(id), 10)
	}
	return bf.AddMulti(ctx, items)
}

// ExistsIDs 批量检查数字ID是否存在于布隆过滤器中
func (bf *BloomFilter) ExistsIDs(ctx context.Context, ids []uint) ([]bool, error) {
	items := make([]string, len(ids))
	for i, id := range ids {
		items[i] = strconv.FormatUint(uint64(id), 10)
	}
	return bf.ExistsMulti(ctx, items)
}

// 预定义配置
var (
	// ShopBloomConfig 商铺布隆过滤器配置
	ShopBloomConfig = BloomFilterConfig{
		Key:       "shop:bloom:filter",
		ErrorRate: 0.01,   // 1%误判率
		Capacity:  100000, // 10万商铺容量
		Expansion: 2,      // 2倍扩容
	}

	// UserBloomConfig 用户布隆过滤器配置
	UserBloomConfig = BloomFilterConfig{
		Key:       "user:bloom:filter",
		ErrorRate: 0.001,   // 0.1%误判率
		Capacity:  1000000, // 100万用户容量
		Expansion: 2,       // 2倍扩容
	}

	// VoucherBloomConfig 优惠券布隆过滤器配置
	VoucherBloomConfig = BloomFilterConfig{
		Key:       "voucher:bloom:filter",
		ErrorRate: 0.01,  // 1%误判率
		Capacity:  50000, // 5万优惠券容量
		Expansion: 2,     // 2倍扩容
	}
)

// CreateShopBloomFilter 创建商铺布隆过滤器
func CreateShopBloomFilter(rdb *redis.Client) *BloomFilter {
	return NewBloomFilter(rdb, ShopBloomConfig)
}

// CreateUserBloomFilter 创建用户布隆过滤器
func CreateUserBloomFilter(rdb *redis.Client) *BloomFilter {
	return NewBloomFilter(rdb, UserBloomConfig)
}

// CreateVoucherBloomFilter 创建优惠券布隆过滤器
func CreateVoucherBloomFilter(rdb *redis.Client) *BloomFilter {
	return NewBloomFilter(rdb, VoucherBloomConfig)
}

// IDProvider ID提供器接口，用于解耦数据库依赖
type IDProvider interface {
	GetAllShopIDs(ctx context.Context) ([]uint, error)
	GetAllUserIDs(ctx context.Context) ([]uint, error)
	GetAllVoucherIDs(ctx context.Context) ([]uint, error)
}

// IDProviderFunc 函数适配器，用于将函数转换为IDProvider接口
type IDProviderFunc struct {
	ShopIDsFunc    func(context.Context) ([]uint, error)
	UserIDsFunc    func(context.Context) ([]uint, error)
	VoucherIDsFunc func(context.Context) ([]uint, error)
}

// GetAllShopIDs 获取所有商铺ID
func (f IDProviderFunc) GetAllShopIDs(ctx context.Context) ([]uint, error) {
	if f.ShopIDsFunc != nil {
		return f.ShopIDsFunc(ctx)
	}
	return nil, nil
}

// GetAllUserIDs 获取所有用户ID
func (f IDProviderFunc) GetAllUserIDs(ctx context.Context) ([]uint, error) {
	if f.UserIDsFunc != nil {
		return f.UserIDsFunc(ctx)
	}
	return nil, nil
}

// GetAllVoucherIDs 获取所有优惠券ID
func (f IDProviderFunc) GetAllVoucherIDs(ctx context.Context) ([]uint, error) {
	if f.VoucherIDsFunc != nil {
		return f.VoucherIDsFunc(ctx)
	}
	return nil, nil
}

// BloomInitializer 布隆过滤器初始化器
type BloomInitializer struct {
	rdb         *redis.Client
	idProvider  IDProvider
	mu          sync.RWMutex    // 保护初始化状态
	initialized map[string]bool // 记录各过滤器初始化状态
}

// NewBloomInitializer 创建布隆过滤器初始化器
// 使用接口解耦数据库依赖，便于测试和替换实现
func NewBloomInitializer(rdb *redis.Client, provider IDProvider) *BloomInitializer {
	return &BloomInitializer{
		rdb:         rdb,
		idProvider:  provider,
		initialized: make(map[string]bool),
	}
}

// InitShopBloomFilter 初始化商铺布隆过滤器
func (bi *BloomInitializer) InitShopBloomFilter(ctx context.Context) error {
	if bi.idProvider == nil {
		return fmt.Errorf("IDProvider未设置")
	}
	return bi.initBloomFilterWithIDs(ctx, ShopBloomConfig, bi.idProvider.GetAllShopIDs)
}

// InitUserBloomFilter 初始化用户布隆过滤器
func (bi *BloomInitializer) InitUserBloomFilter(ctx context.Context) error {
	if bi.idProvider == nil {
		return fmt.Errorf("IDProvider未设置")
	}
	return bi.initBloomFilterWithIDs(ctx, UserBloomConfig, bi.idProvider.GetAllUserIDs)
}

// InitVoucherBloomFilter 初始化优惠券布隆过滤器
func (bi *BloomInitializer) InitVoucherBloomFilter(ctx context.Context) error {
	if bi.idProvider == nil {
		return fmt.Errorf("IDProvider未设置")
	}
	return bi.initBloomFilterWithIDs(ctx, VoucherBloomConfig, bi.idProvider.GetAllVoucherIDs)
}

// initBloomFilterWithIDs 通用的布隆过滤器初始化方法
func (bi *BloomInitializer) initBloomFilterWithIDs(ctx context.Context, config BloomFilterConfig, getIDsFunc func(context.Context) ([]uint, error)) error {
	if getIDsFunc == nil {
		return fmt.Errorf("getIDsFunc不能为空")
	}

	// 创建布隆过滤器
	bf := NewBloomFilter(bi.rdb, config)

	// 创建或重置布隆过滤器
	if err := bf.Reserve(ctx); err != nil {
		return fmt.Errorf("创建布隆过滤器失败: %v", err)
	}

	// 获取所有ID
	ids, err := getIDsFunc(ctx)
	if err != nil {
		return fmt.Errorf("获取ID列表失败: %v", err)
	}

	if len(ids) == 0 {
		log.Printf("警告: %s 没有找到任何数据", config.Key)
		return nil
	}

	// 分批处理大数据量，每批10000个
	const batchSize = 10000
	totalAdded := 0

	for i := 0; i < len(ids); i += batchSize {
		end := i + batchSize
		if end > len(ids) {
			end = len(ids)
		}
		batch := ids[i:end]

		// 批量添加ID到布隆过滤器
		results, err := bf.AddIDs(ctx, batch)
		if err != nil {
			return fmt.Errorf("批量添加ID到布隆过滤器失败(批次%d-%d): %v", i, end, err)
		}

		// 统计添加结果
		for _, added := range results {
			if added {
				totalAdded++
			}
		}

		log.Printf("布隆过滤器 %s 批次处理: %d/%d", config.Key, end, len(ids))
	}

	// 更新初始化状态
	bi.mu.Lock()
	bi.initialized[config.Key] = true
	bi.mu.Unlock()

	log.Printf("布隆过滤器 %s 初始化完成: 总数据量=%d, 新增=%d", config.Key, len(ids), totalAdded)
	return nil
}

// InitAllBloomFilters 初始化所有布隆过滤器
func (bi *BloomInitializer) InitAllBloomFilters(ctx context.Context) error {
	if bi.idProvider == nil {
		return fmt.Errorf("IDProvider未设置")
	}

	log.Println("开始初始化所有布隆过滤器...")
	start := time.Now()

	// 使用WaitGroup并发初始化
	var wg sync.WaitGroup
	errChan := make(chan error, 3)

	// 初始化商铺布隆过滤器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bi.InitShopBloomFilter(ctx); err != nil {
			log.Printf("初始化商铺布隆过滤器失败: %v", err)
			errChan <- fmt.Errorf("商铺布隆过滤器: %v", err)
		}
	}()

	// 初始化用户布隆过滤器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bi.InitUserBloomFilter(ctx); err != nil {
			log.Printf("初始化用户布隆过滤器失败: %v", err)
			errChan <- fmt.Errorf("用户布隆过滤器: %v", err)
		}
	}()

	// 初始化优惠券布隆过滤器
	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := bi.InitVoucherBloomFilter(ctx); err != nil {
			log.Printf("初始化优惠券布隆过滤器失败: %v", err)
			errChan <- fmt.Errorf("优惠券布隆过滤器: %v", err)
		}
	}()

	wg.Wait()
	close(errChan)

	// 检查是否有错误
	var errs []error
	for err := range errChan {
		errs = append(errs, err)
	}

	if len(errs) > 0 {
		return fmt.Errorf("部分布隆过滤器初始化失败: %v", errs)
	}

	duration := time.Since(start)
	log.Printf("所有布隆过滤器初始化完成，耗时: %v", duration)
	return nil
}

// CheckBloomFilterHealth 检查布隆过滤器健康状态
func (bi *BloomInitializer) CheckBloomFilterHealth(ctx context.Context) map[string]interface{} {
	health := make(map[string]interface{})

	configs := []BloomFilterConfig{ShopBloomConfig, UserBloomConfig, VoucherBloomConfig}

	for _, config := range configs {
		bf := NewBloomFilter(bi.rdb, config)
		info, err := bf.Info(ctx)
		if err != nil {
			health[config.Key] = map[string]interface{}{
				"status": "error",
				"error":  err.Error(),
			}
		} else {
			health[config.Key] = map[string]interface{}{
				"status": "healthy",
				"info":   info,
			}
		}
	}

	return health
}

// AddToBloomFilter 向指定布隆过滤器添加新ID（用于增量更新）
func (bi *BloomInitializer) AddToBloomFilter(ctx context.Context, filterType string, id uint) error {
	var config BloomFilterConfig

	switch filterType {
	case "shop":
		config = ShopBloomConfig
	case "user":
		config = UserBloomConfig
	case "voucher":
		config = VoucherBloomConfig
	default:
		return fmt.Errorf("不支持的过滤器类型: %s", filterType)
	}

	bf := NewBloomFilter(bi.rdb, config)
	_, err := bf.AddID(ctx, id)
	if err != nil {
		return fmt.Errorf("添加ID到布隆过滤器失败: %v", err)
	}

	log.Printf("成功添加ID %d 到布隆过滤器 %s", id, config.Key)
	return nil
}

// CheckIDExists 检查ID是否存在于指定布隆过滤器中
func (bi *BloomInitializer) CheckIDExists(ctx context.Context, filterType string, id uint) (bool, error) {
	var config BloomFilterConfig

	switch filterType {
	case "shop":
		config = ShopBloomConfig
	case "user":
		config = UserBloomConfig
	case "voucher":
		config = VoucherBloomConfig
	default:
		return false, fmt.Errorf("不支持的过滤器类型: %s", filterType)
	}

	bf := NewBloomFilter(bi.rdb, config)
	return bf.ExistsID(ctx, id)
}

// CheckIDExists 检查ID是否存在于指定布隆过滤器中（全局函数）
// 注意：此函数需要全局Redis连接，建议使用CheckStringExistsInBloomFilter
func CheckIDExists(filterType string, id uint) bool {
	// 这里需要获取Redis连接，暂时返回true避免阻塞
	// 在实际使用中，应该传入Redis连接或使用全局连接
	return true
}

// CheckIDExistsWithRedis 检查ID是否存在于指定布隆过滤器中（带Redis连接）
func CheckIDExistsWithRedis(ctx context.Context, rdb *redis.Client, filterType string, id uint) (bool, error) {
	var key string

	switch filterType {
	case "shop":
		key = ShopBloomConfig.Key
	case "user":
		key = UserBloomConfig.Key
	case "voucher":
		key = VoucherBloomConfig.Key
	default:
		return false, fmt.Errorf("不支持的过滤器类型: %s", filterType)
	}

	// 将ID转换为字符串
	idStr := strconv.FormatUint(uint64(id), 10)

	// 使用通用函数检查
	return CheckStringExistsInBloomFilter(ctx, rdb, key, idStr)
}

// CheckStringExistsInBloomFilter 通用函数：检查字符串是否存在于指定key的布隆过滤器中
// 支持降级机制：当Redis故障时返回true（放行），避免阻塞正常业务流程
func CheckStringExistsInBloomFilter(ctx context.Context, rdb *redis.Client, key string, value string) (bool, error) {
	if rdb == nil {
		// 降级：Redis未配置时放行
		log.Printf("[BloomFilter降级] Redis客户端为空，放行key=%s", key)
		return true, nil
	}

	if key == "" {
		return false, fmt.Errorf("布隆过滤器key不能为空")
	}

	if value == "" {
		return false, fmt.Errorf("检查的值不能为空")
	}

	// 直接使用Redis命令检查元素是否存在
	// BF.EXISTS 返回整数 0 或 1，使用 Result() 然后手动转换
	cmd := rdb.Do(ctx, "BF.EXISTS", key, value)
	if err := cmd.Err(); err != nil {
		// 降级处理：Redis故障时返回true，放行请求
		log.Printf("[BloomFilter降级] 检查失败: %v, key=%s, value=%s", err, key, value)
		return true, nil
	}

	// 解析结果：BF.EXISTS 返回 int64 类型的 0 或 1
	result, err := cmd.Result()
	if err != nil {
		log.Printf("[BloomFilter降级] 解析结果失败: %v, key=%s, value=%s", err, key, value)
		return true, nil
	}

	// 处理不同的返回类型
	switch v := result.(type) {
	case int64:
		return v == 1, nil
	case int:
		return v == 1, nil
	case bool:
		return v, nil
	default:
		log.Printf("[BloomFilter] 未知的返回类型: %T, value=%v", result, result)
		return true, nil // 降级放行
	}
}

// NewBloomFilterWithFallback 创建带降级机制的布隆过滤器包装器
func NewBloomFilterWithFallback(bf *BloomFilter, onError func(error)) *BloomFilterWithFallback {
	return &BloomFilterWithFallback{
		bf:      bf,
		stats:   &BloomFilterStats{},
		onError: onError,
	}
}

// Exists 带降级机制的存在性检查
func (bff *BloomFilterWithFallback) Exists(ctx context.Context, item string) bool {
	exists, err := bff.bf.Exists(ctx, item)
	if err != nil {
		bff.mu.Lock()
		bff.stats.ErrorCount++
		bff.stats.LastErrorTime = time.Now()
		bff.stats.IsDegraded = true
		bff.mu.Unlock()

		if bff.onError != nil {
			bff.onError(err)
		}

		// 降级：错误时返回true（放行）
		log.Printf("[BloomFilter降级] Exists错误: %v, 放行item=%s", err, item)
		return true
	}

	bff.mu.Lock()
	bff.stats.TotalQueries++
	if exists {
		bff.stats.HitCount++
	} else {
		bff.stats.MissCount++
	}
	bff.stats.IsDegraded = false
	bff.mu.Unlock()

	return exists
}

// ExistsID 带降级机制的数字ID检查
func (bff *BloomFilterWithFallback) ExistsID(ctx context.Context, id uint) bool {
	return bff.Exists(ctx, strconv.FormatUint(uint64(id), 10))
}

// GetStats 获取统计信息
func (bff *BloomFilterWithFallback) GetStats() BloomFilterStats {
	bff.mu.RLock()
	defer bff.mu.RUnlock()
	return *bff.stats
}

// IsHealthy 检查布隆过滤器是否健康
func (bff *BloomFilterWithFallback) IsHealthy() bool {
	bff.mu.RLock()
	defer bff.mu.RUnlock()
	return !bff.stats.IsDegraded
}
