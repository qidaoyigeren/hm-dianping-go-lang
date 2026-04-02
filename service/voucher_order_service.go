package service

import (
	"context"
	"encoding/json"
	"fmt"
	"hm-dianping-go/dao"
	"hm-dianping-go/models"
	"hm-dianping-go/utils"
	"log"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/segmentio/kafka-go"
)

// SeckillVoucher 秒杀优惠券
func SeckillVoucher(ctx context.Context, userId, voucherId uint) *utils.Result {
	// 从文件当中加载脚本
	script, err := os.ReadFile("script/seckill.lua")
	if err != nil {
		log.Printf("读取秒杀脚本失败: %v", err)
		return utils.ErrorResult("系统错误")
	}
	scriptStr := string(script)

	// 生成订单ID
	orderID := strconv.Itoa(int(time.Now().UnixNano()))

	// 1. 执行Lua脚本
	result := dao.Redis.Eval(ctx, scriptStr, []string{}, strconv.Itoa(int(voucherId)), strconv.Itoa(int(userId)), orderID)
	if result.Err() != nil {
		log.Printf("执行秒杀脚本失败: %v", result.Err())
		return utils.ErrorResult("系统错误")
	}

	// 2. 判断结果是否为 0，0的时候有资格完成
	r, err := result.Int()
	if err != nil {
		log.Printf("获取秒杀脚本返回值失败: %v", err)
		return utils.ErrorResult("系统错误")
	}
	if r != 0 {
		if r == 1 {
			return utils.ErrorResult("库存不足")
		}
		return utils.ErrorResult("不能重复购买")
	}

	// 3. 发送订单到Kafka
	err = SendOrderToKafka(ctx, userId, voucherId, orderID)
	if err != nil {
		log.Printf("发送订单到Kafka失败: %v", err)
		return utils.ErrorResult("系统繁忙，请稍后重试")
	}

	// 4. 返回订单ID（这里可以生成一个临时ID或者返回成功信息）
	return utils.SuccessResultWithData("秒杀成功，订单处理中...")
}

// KafkaOrderInfo Kafka中的订单信息结构体
type KafkaOrderInfo struct {
	UserID    string `json:"userId"`
	VoucherID string `json:"voucherId"`
	OrderID   string `json:"id"`
}

// Kafka相关配置
var (
	kafkaTopic  = "voucher-orders"            // Kafka主题
	kafkaBroker = "host.docker.internal:9092" // Kafka broker地址（Docker Desktop使用host.docker.internal）
	//kafkaBroker    = "localhost:9092" // 校园网环境下使用
	kafkaGroupID   = "voucher-order-group" // Kafka 消费者组 ID
	consumerCount  = 3                     // 消费者数量
	kafkaOnce      sync.Once               // 确保Kafka只初始化一次
	kafkaWriter    *kafka.Writer           // Kafka写入器
	kafkaConsumers []*kafka.Reader         // Kafka消费者
	stopChan       = make(chan struct{})   // 停止信号
	wg             sync.WaitGroup          // 等待组，用于优雅关闭
)

// InitKafkaConsumer 初始化Kafka消费者
func InitKafkaConsumer() error {
	var initErr error
	kafkaOnce.Do(func() {
		// 1. 初始化Kafka写入器
		kafkaWriter = &kafka.Writer{
			Addr:     kafka.TCP(kafkaBroker),
			Topic:    kafkaTopic,
			Balancer: &kafka.LeastBytes{},
		}

		// 2. 初始化Kafka消费者
		kafkaConsumers = make([]*kafka.Reader, consumerCount)
		for i := 0; i < consumerCount; i++ {
			kafkaConsumers[i] = kafka.NewReader(kafka.ReaderConfig{
				Brokers:         []string{kafkaBroker},
				Topic:           kafkaTopic,
				GroupID:         kafkaGroupID,
				MinBytes:        10e3,        // 10KB
				MaxBytes:        10e6,        // 10MB
				MaxWait:         time.Second, // 最多等待1秒
				ReadLagInterval: time.Second,
			})

			// 启动消费者
			wg.Add(1)
			go kafkaConsumer(kafkaConsumers[i], i)
		}

		log.Printf("Kafka消费者初始化完成，Topic: %s, 消费者数量: %d",
			kafkaTopic, consumerCount)
	})

	return initErr
}

// kafkaConsumer Kafka消费者worker
func kafkaConsumer(reader *kafka.Reader, workerID int) {
	defer wg.Done()
	defer reader.Close()

	log.Printf("Kafka消费者 (Worker %d) 启动", workerID)

	for {
		select {
		case <-stopChan:
			log.Printf("Kafka消费者 (Worker %d) 收到停止信号，正在退出", workerID)
			return
		default:
			// 使用带超时的上下文，避免长时间阻塞
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

			// 从Kafka中读取消息
			msg, err := reader.ReadMessage(ctx)
			cancel() // 立即释放上下文

			if err != nil {
				// 检查是否是超时错误（正常情况，没有新消息）
				if err == context.DeadlineExceeded {
					// 超时是正常现象，静默继续循环检查 stopChan
					continue
				}
				// 检查是否是停止信号导致的错误
				select {
				case <-stopChan:
					log.Printf("Kafka消费者 (Worker %d) 收到停止信号，正在退出", workerID)
					return
				default:
					// 只打印非超时错误
					log.Printf("消费者 %d 读取消息失败: %v", workerID, err)
					time.Sleep(time.Second * 5) // 出错时等待2秒再重试
				}
				continue
			}

			// 处理消息
			err = processKafkaMessage(context.Background(), msg, workerID)
			if err != nil {
				log.Printf("消费者 %d 处理消息失败：offset=%d, error=%v",
					workerID, msg.Offset, err)
				// 处理失败时不提交 offset，下次会继续消费这条消息
				// 这里可以添加重试逻辑或将失败消息放入死信队列
			} else {
				log.Printf("消费者 %d 成功处理消息：offset=%d", workerID, msg.Offset)
				// 处理成功后提交 offset
				if err := reader.CommitMessages(context.Background(), msg); err != nil {
					log.Printf("消费者 %d 提交 offset 失败：%v", workerID, err)
				}
			}
		}
	}
}

// processKafkaMessage 处理单条Kafka消息
func processKafkaMessage(ctx context.Context, msg kafka.Message, workerID int) error {
	// 解析消息内容
	orderInfo, err := parseKafkaMessage(msg.Value)
	if err != nil {
		return fmt.Errorf("解析消息失败: %v", err)
	}

	// 转换字符串ID为uint
	userID, err := strconv.ParseUint(orderInfo.UserID, 10, 32)
	if err != nil {
		return fmt.Errorf("解析用户ID失败: %v", err)
	}

	voucherID, err := strconv.ParseUint(orderInfo.VoucherID, 10, 32)
	if err != nil {
		return fmt.Errorf("解析优惠券ID失败: %v", err)
	}

	// 处理订单
	return processKafkaOrder(ctx, uint(userID), uint(voucherID), orderInfo.OrderID)
}

// parseKafkaMessage 解析Kafka订单消息
func parseKafkaMessage(data []byte) (*KafkaOrderInfo, error) {
	var orderInfo KafkaOrderInfo
	if err := json.Unmarshal(data, &orderInfo); err != nil {
		return nil, fmt.Errorf("解析消息失败: %v", err)
	}

	// 验证必要字段
	if orderInfo.UserID == "" {
		return nil, fmt.Errorf("消息中缺少userId字段")
	}
	if orderInfo.VoucherID == "" {
		return nil, fmt.Errorf("消息中缺少voucherId字段")
	}

	return &orderInfo, nil
}

// processKafkaOrder 处理Kafka中的订单（幂等性：同一订单ID只处理一次）
func processKafkaOrder(ctx context.Context, userID, voucherID uint, orderID string) error {
	// 幂等性检查：如果该订单已存在，直接返回成功
	// 使用Redis记录已处理的订单ID（5分钟过期，防止内存无限增长）
	processedKey := "kafka:processed:order:" + orderID
	exists, err := dao.Redis.Exists(ctx, processedKey).Result()
	if err != nil {
		log.Printf("幂等性检查失败: %v", err)
		// 检查失败继续处理，不阻塞
	} else if exists > 0 {
		log.Printf("订单已处理过，跳过: orderID=%s", orderID)
		return nil
	}

	// 开始数据库事务
	tx := dao.DB.Begin()
	if tx.Error != nil {
		return fmt.Errorf("开始事务失败: %v", tx.Error)
	}

	defer func() {
		if r := recover(); r != nil {
			tx.Rollback()
			log.Printf("订单处理发生panic: %v", r)
		}
	}()

	// 创建订单
	now := time.Now()
	order := &models.VoucherOrder{
		UserID:      userID,
		VoucherID:   voucherID,
		PayType:     1,
		Status:      1,
		CreateTime:  &now,
		VoucherType: 2, // 秒杀券类型
	}

	// 创建订单记录
	if err := dao.CreateVoucherOrder(ctx, tx, order); err != nil {
		tx.Rollback()
		return fmt.Errorf("创建订单失败：%v", err)
	}

	// 扣减数据库中的秒杀券库存
	if err := dao.UpdateSeckillVoucherStockDB(tx, voucherID, 1); err != nil {
		tx.Rollback()
		return fmt.Errorf("扣减库存失败：%v", err)
	}

	// 提交事务
	if err := tx.Commit().Error; err != nil {
		tx.Rollback()
		return fmt.Errorf("提交事务失败：%v", err)
	}

	// 标记订单已处理（5分钟过期）
	if err := dao.Redis.Set(ctx, processedKey, "1", 5*time.Minute).Err(); err != nil {
		log.Printf("标记订单已处理失败: %v", err)
		// 标记失败不影响主流程
	}

	log.Printf("成功创建订单: userID=%d, voucherID=%d, orderID=%s", userID, voucherID, orderID)

	return nil
}

// StopKafkaConsumers 停止所有Kafka消费者（用于优雅关闭）
func StopKafkaConsumers() {
	log.Println("正在停止Kafka消费者...")
	close(stopChan)
	wg.Wait()
	if kafkaWriter != nil {
		kafkaWriter.Close()
	}
	log.Println("所有Kafka消费者已停止")
}

// SendOrderToKafka 发送订单到Kafka（带重试机制）
func SendOrderToKafka(ctx context.Context, userID, voucherID uint, orderID string) error {
	if kafkaWriter == nil {
		return fmt.Errorf("Kafka写入器未初始化")
	}

	orderInfo := KafkaOrderInfo{
		UserID:    strconv.Itoa(int(userID)),
		VoucherID: strconv.Itoa(int(voucherID)),
		OrderID:   orderID,
	}

	data, err := json.Marshal(orderInfo)
	if err != nil {
		return fmt.Errorf("序列化订单信息失败: %v", err)
	}

	msg := kafka.Message{
		Key:   []byte(orderID), // 使用orderID作为key，确保相同订单路由到同一分区
		Value: data,
	}

	// 重试机制：最多3次，间隔递增
	var lastErr error
	for i := 0; i < 3; i++ {
		if i > 0 {
			backoff := time.Duration(i*100) * time.Millisecond
			log.Printf("Kafka发送重试第%d次，等待%v...", i, backoff)
			time.Sleep(backoff)
		}

		err = kafkaWriter.WriteMessages(ctx, msg)
		if err == nil {
			return nil // 发送成功
		}

		lastErr = err
		log.Printf("Kafka发送失败(尝试%d/3): %v", i+1, err)
	}

	return fmt.Errorf("Kafka发送失败，已重试3次: %v", lastErr)
}
