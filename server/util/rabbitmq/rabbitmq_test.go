package rabbitmq_test

import (
	"testing"

	"git.app.uib.no/hvl-dat-250/26h/semester-project/group-8/learning-platform/util/rabbitmq"
	"github.com/stretchr/testify/assert"
)

func TestDeadLetterQueueName(t *testing.T) {
	assert.Equal(t, "submission.created.dead", rabbitmq.DeadLetterQueueName("submission.created"))
	assert.Equal(t, "events.dead", rabbitmq.DeadLetterQueueName("events"))
}

func TestQueueArgsWithDLQ(t *testing.T) {
	args := rabbitmq.QueueArgsWithDLQ("submission.created")
	assert.Equal(t, "", args["x-dead-letter-exchange"])
	assert.Equal(t, "submission.created.dead", args["x-dead-letter-routing-key"])
}

func TestNewClient_InvalidUrlFailsGracefully(t *testing.T) {
	// Connect to non-existent port to test fail and backoff
	client, err := rabbitmq.NewClient("amqp://invalid-host-that-does-not-exist:12345/")
	assert.Error(t, err)
	assert.Nil(t, client)
}
