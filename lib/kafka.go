package julive_com

import (
	"../kafka"
	"strconv"
)

var (
	client      sarama.SyncProducer
	kafkaSender *KafkaSender
)

type Message struct {
	line  string
	topic string
}

type KafkaSender struct {
	client   sarama.SyncProducer
	lineChan chan string
}

type KafkaConfType struct {
	KafkaHost      string
	KafkaTopic     string
	KafkaPort      int
	KafkaChan      int
	KafkaThreadNum int
}

// kafka
func NewKafkaSender(kafka_conf KafkaConfType) (kafka *KafkaSender, err error) {
	kafkaconfig := kafka_conf.KafkaHost + ":" + strconv.Itoa(kafka_conf.KafkaPort)
	kafka = &KafkaSender{
		lineChan: make(chan string, kafka_conf.KafkaChan),
	}
	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll
	config.Producer.Partitioner = sarama.NewRandomPartitioner
	config.Producer.Return.Successes = true
	client, err := sarama.NewSyncProducer([]string{kafkaconfig}, config)
	if err != nil {
		Error.Println("init kafka Producer client failed,err:%v\n", err)
	}
	kafka.client = client
	// for i := 0; i < kafka_conf.KafkaThreadNum; i++ {
	// 	// 根据配置文件循环开启线程去发消息到kafka
	// 	go kafka.sendToKafka(kafka_conf.KafkaTopic)
	// }
	return kafka, err
}

func (k *KafkaSender) sendToKafka(toppic string) {
	//从channel中读取日志内容放到kafka消息队列中
	for v := range k.lineChan {
		msg := &sarama.ProducerMessage{}
		msg.Topic = toppic
		msg.Value = sarama.StringEncoder(v)
		_, _, err := k.client.SendMessage(msg)
		if err != nil {
			Error.Println("send message to kafka failed,err:%v", err)
		}
	}
}

func NewMessage(k *KafkaSender, toppic string, line string) (err error) {
	// k.lineChan <- line
	msg := &sarama.ProducerMessage{}
	msg.Topic = toppic
	msg.Value = sarama.StringEncoder(line)
	partition, offset, err := k.client.SendMessage(msg)
	if err != nil {
		Error.Println("Failed to produce message :%s", err)
	}
	Info.Println("partition:%d, offset: %d\n", partition, offset)
	defer k.client.Close()
	return
}

func NewReciver(kafka_conf KafkaConfType) sarama.Consumer {
	kafkaconfig := kafka_conf.KafkaHost + ":" + strconv.Itoa(kafka_conf.KafkaPort)
	config := sarama.NewConfig()

	consumer, err := sarama.NewConsumer([]string{kafkaconfig}, config)
	if err != nil {
		Error.Println("init kafka Consumer client failed,err:%v\n", err)
	}
	defer consumer.Close()
	return consumer
}

func GetPartitionList(kafka_conf KafkaConfType, consumer sarama.Consumer) ([]int32, error) {
	partitionList, err := consumer.Partitions(kafka_conf.KafkaTopic)
	if err != nil {
		Error.Println("get kafka partitionList failed,err:%v\n", err)
	}
	return partitionList, err
}

func NewPartition(kafka_conf KafkaConfType, partition int32, consumer sarama.Consumer) sarama.PartitionConsumer {
	partitionConsumer, err := consumer.ConsumePartition(kafka_conf.KafkaTopic, partition, sarama.OffsetNewest)
	if err != nil {
		Error.Println("init kafka Partition failed,err:%v\n", err)
	}
	defer partitionConsumer.Close()
	//循环等待接受消息.
	for {
		select {
		//接收消息通道和错误通道的内容.
		case msg := <-partitionConsumer.Messages():
			Info.Println("msg offset: ", msg.Offset, " partition: ", msg.Partition, " timestrap: ", msg.Timestamp.Format("2006-Jan-02 15:04"), " value: ", string(msg.Value))
		case err := <-partitionConsumer.Errors():
			Info.Println(err.Err)
		}
	}
	return partitionConsumer
}
