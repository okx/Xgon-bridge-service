package coinmiddleware

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"github.com/0xPolygonHermez/zkevm-bridge-service/redisstorage"
	"github.com/0xPolygonHermez/zkevm-node/log"
	"github.com/IBM/sarama"
	"github.com/pkg/errors"
)

// KafkaConsumer provides the interface to consume from coin middleware kafka
type KafkaConsumer interface {
	Start(ctx context.Context) error
	Close() error
}

type kafkaConsumerImpl struct {
	topics  []string
	client  sarama.ConsumerGroup
	handler sarama.ConsumerGroupHandler
}

// aliyun root ca
var rootCA = `-----BEGIN CERTIFICATE-----
MIIFKjCCAxICCQCdkV+iL/cBTzANBgkqhkiG9w0BAQsFADBWMQswCQYDVQQGEwJD
TjEQMA4GA1UECAwHQmVpamluZzEQMA4GA1UEBwwHQmVpamluZzEQMA4GA1UECgwH
QWxpYmFiYTERMA8GA1UEAwwIQWxpS2Fma2EwIBcNMjIwNTExMTAzOTMxWhgPMjEy
MjA0MTcxMDM5MzFaMFYxCzAJBgNVBAYTAkNOMRAwDgYDVQQIDAdCZWlqaW5nMRAw
DgYDVQQHDAdCZWlqaW5nMRAwDgYDVQQKDAdBbGliYWJhMREwDwYDVQQDDAhBbGlL
YWZrYTCCAiIwDQYJKoZIhvcNAQEBBQADggIPADCCAgoCggIBAL315apERcpAkDAB
SY4A2bGrRZO4CXj4nvqbwEZ50f1HlwABjzUMKXES7lWrOwrnqZjSIgm5woqu+Pr4
sWhKFHN19SSnjeKilQoL8SzMk0p22QJK2sqKRMuHtoBtL6uOT+ykV16IEg0fY2Uu
/oX/sF2LAVCIl1IGc2HVKUr56c0/mM6V6Ur5Sum7ctKk2dm6YS5gwDOXcqAaZhwd
jVzqLEW8hmsMS7n+d2/NIJMqXvTHDRQ74xhR9tN2w92keEBGOQoMG/Qw0RvS1aQi
RKpNpvCE7z543istYuFbFji646u6kRCr7I2i4RwV0qXVM1djcS+PysUsIX4mEjdP
Kq0Fptzsii3aeTFuNswOlo5GieE3psVoymIP2HWd6xmlmFaX3Z8Nd4PxA6h0uRIY
tRbLkHw8WfOAl4dXxWQFkbvNYNLRB5xZUYjm3CA+ZhYfJRtNlPa2247Psnbup6CH
k3DP+aExdLmbtyugZO/lNqi9WMZ0qLFGXZDz8astgJPGKiCjihccpP1cdzGlCzGu
iE6S25JEBuXPl4wg4GXNuCg6tcEKL2qinvbrCimrilWuFajBh7hRH0dgkhezw6xU
+3++ZCebEJOXZ8byn3v/gmyx2PDnKlBPcXCy23nadbiX/zpNvNvCqAewajm9AlWY
fXbCl5TkUnyMPsh0rwWeeRYR2kM3AgMBAAEwDQYJKoZIhvcNAQELBQADggIBADxW
YJoWh9DVtwFGp8TOrlbZ7kwflKFv8Hew4SX00K5GwKgmnn3fjdR0F8rZ2ar/BqdD
zR63sv9LGjMci9NWAqPqN5MyKB97KrFV6nHzcYLRmT+ltolqcfp5MeGCqka7ZTEL
t658xxaSXNEY9HGHYskIu7mWd41KAj0RLRJnEEOrCSZzfpzG4LdD6J0u7wpyJSYL
jGxi2xswt5C0x790LS/JmFq65c/vzfATjbmu6XSO3UvtsADpj0pH3FJFhLzoT67o
NrUeFEHrzsMc7JenYmPIYmEb4xXlfctjCzLaiNG3u8uKwXGBk/oagAwXCsI8I0pR
wtW/QedXxlFtUfATRZnI/eLqvJ5cQ6aXg/GyJtAv+ccFf004K1ER00ECe738WNXm
+6NNkhN5gPhwsfoDhq+a7Zmvj9+x/XDjSRqZ8j+XIMi9ZQjTwUAg9JmnhyR4eJXn
oQAxGc3ii98YoAspKZGRX6LoRfYbNE3TXJsSzGw73+PqS1y74xNNmMx2XX6IV/53
Is5mA8fli6BIEKkAgE6Pn0t6v5EP6haVF84vJazYRIlYflR2mi8p8dU6kohiC79C
e4seRTTZgyXU+5dgFIXqagub2A79tRtPAr+4Xi84jzY84ceUwqX2fxRwkfaUUJb8
Hh2q+P+VJeK50B83DZ4ui+WNJbAaAbcLMsn/idX3
-----END CERTIFICATE-----`

func NewKafkaConsumer(cfg Config, redisStorage redisstorage.RedisStorage) (KafkaConsumer, error) {
	config := sarama.NewConfig()
	config.Consumer.Offsets.Initial = cfg.InitialOffset

	// Enable SASL authentication
	if cfg.Username != "" && cfg.Password != "" {
		config.Net.SASL.Enable = true
		config.Net.SASL.User = cfg.Username
		config.Net.SASL.Password = cfg.Password

		caCertPool := x509.NewCertPool()
		if ok := caCertPool.AppendCertsFromPEM([]byte(rootCA)); !ok {
			return nil, errors.New("NewKafkaConsumer caCertPool.AppendCertsFromPEM")
		}

		config.Net.TLS.Enable = true
		config.Net.TLS.Config = &tls.Config{RootCAs: caCertPool, InsecureSkipVerify: true}
	}

	client, err := sarama.NewConsumerGroup(cfg.Brokers, cfg.ConsumerGroupID, config)
	if err != nil {
		return nil, errors.Wrap(err, "kafka consumer group init error")
	}

	return &kafkaConsumerImpl{
		topics:  cfg.Topics,
		client:  client,
		handler: NewMessageHandler(redisStorage),
	}, nil
}

func (c *kafkaConsumerImpl) Start(ctx context.Context) error {
	log.Debug("starting kafka consumer")
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	for {
		log.Debugf("start consume")
		err := c.client.Consume(ctx, c.topics, c.handler)
		if err != nil {
			log.Errorf("kafka consumer error: %v", err)
			if errors.Is(err, sarama.ErrClosedConsumerGroup) {
				err = nil
			}
			return errors.Wrap(err, "kafka consumer error")
		}
		if err = ctx.Err(); err != nil {
			log.Errorf("kafka consumer ctx error: %v", err)
			return errors.Wrap(err, "kafka consumer ctx error")
		}
	}
}

func (c *kafkaConsumerImpl) Close() error {
	log.Debug("closing kafka consumer...")
	return c.client.Close()
}