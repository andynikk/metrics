package rabbitmq

import (
	"context"
	"time"

	"github.com/andynikk/advancedmetrics/internal/constants"
	amqp "github.com/rabbitmq/amqp091-go"
)

type SettingRMQ struct {
	Conn    *amqp.Connection
	Channel *amqp.Channel
	Queue   amqp.Queue
}

func (s *SettingRMQ) ConnRMQ() {
	conn, err := amqp.Dial("amqp://guest:guest@localhost:5672/")
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	s.Conn = conn

}

func (s *SettingRMQ) ChannelRMQ() {
	ch, err := s.Conn.Channel()
	if err != nil {
		constants.Logger.ErrorLog(err)
	}
	s.Channel = ch
}

func (s *SettingRMQ) QueueRMQ() {
	q, err := s.Channel.QueueDeclare(
		"metrics", // name
		false,     // durable
		false,     // delete when unused
		false,     // exclusive
		false,     // no-wait
		nil,       // arguments
	)
	if err != nil {
		constants.Logger.ErrorLog(err)
	}

	s.Queue = q
}

func (s *SettingRMQ) MessageRMQ(header amqp.Table, msg []byte) bool {

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	err := s.Channel.PublishWithContext(ctx,
		"",           // exchange
		s.Queue.Name, // routing key
		false,        // mandatory
		false,        // immediate
		amqp.Publishing{
			ContentType:     header["Content-Type"].(string),
			ContentEncoding: header["Content-Encoding"].(string),
			Headers:         header,
			Body:            msg,
		})
	if err != nil {
		constants.Logger.ErrorLog(err)
		return false
	}
	return true
}
