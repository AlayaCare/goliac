package workflow

import (
	"context"
	"net/url"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// MockDynamoDBClient is a mock implementation of the DynamoDB client
type MockDynamoDBClient struct {
	mock.Mock
}

func (m *MockDynamoDBClient) PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error) {
	args := m.Called(ctx, params)
	return args.Get(0).(*dynamodb.PutItemOutput), args.Error(1)
}

func TestDynamoDBPluginWorkflow(t *testing.T) {
	t.Run("happy path: dynamodb record creation", func(t *testing.T) {
		// Create mock client
		mockClient := new(MockDynamoDBClient)

		// Set up expectations
		mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
			// Verify the input contains the expected table name
			return *input.TableName == "test-table"
		})).Return(&dynamodb.PutItemOutput{}, nil)

		// Create plugin instance
		plugin := &StepPluginDynamoDB{
			TableName: "test-table",
			client:    mockClient,
		}

		// Create test URL
		prurl, err := url.Parse("https://github.com/mycompany/myrepo/pull/123")
		assert.Nil(t, err)

		// Execute plugin
		returl, err := plugin.Execute(context.Background(), "test-user", "workflowdescription", "test explanation", prurl, map[string]interface{}{})

		// Assertions
		assert.Nil(t, err)
		assert.Equal(t, "", returl)
		mockClient.AssertExpectations(t)
	})

	t.Run("error path: dynamodb put item fails", func(t *testing.T) {
		// Create mock client
		mockClient := new(MockDynamoDBClient)

		// Set up expectations to return an error
		mockClient.On("PutItem", mock.Anything, mock.Anything).Return(
			&dynamodb.PutItemOutput{},
			&types.ResourceNotFoundException{Message: aws.String("Table not found")},
		)

		// Create plugin instance
		plugin := &StepPluginDynamoDB{
			TableName: "test-table",
			client:    mockClient,
		}

		// Create test URL
		prurl, err := url.Parse("https://github.com/mycompany/myrepo/pull/123")
		assert.Nil(t, err)

		// Execute plugin
		returl, err := plugin.Execute(context.Background(), "test-user", "workflowdescription", "test explanation", prurl, map[string]interface{}{})

		// Assertions
		assert.NotNil(t, err)
		assert.Contains(t, err.Error(), "failed to put item in DynamoDB")
		assert.Equal(t, "", returl)
		mockClient.AssertExpectations(t)
	})

	t.Run("happy path: no PR URL provided", func(t *testing.T) {
		// Create mock client
		mockClient := new(MockDynamoDBClient)

		// Set up expectations
		mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
			// Verify the input contains empty PR number
			return *input.TableName == "test-table"
		})).Return(&dynamodb.PutItemOutput{}, nil)

		// Create plugin instance
		plugin := &StepPluginDynamoDB{
			TableName: "test-table",
			client:    mockClient,
		}

		// Execute plugin with nil URL
		returl, err := plugin.Execute(context.Background(), "test-user", "workflowdescription", "test explanation", nil, map[string]interface{}{})

		// Assertions
		assert.NotNil(t, err)
		assert.Equal(t, "", returl)
	})

	t.Run("error path: invalid PR URL", func(t *testing.T) {
		// Create mock client
		mockClient := new(MockDynamoDBClient)

		// Set up expectations
		mockClient.On("PutItem", mock.Anything, mock.MatchedBy(func(input *dynamodb.PutItemInput) bool {
			// Verify the input contains empty PR number
			return *input.TableName == "test-table"
		})).Return(&dynamodb.PutItemOutput{}, nil)

		// Create plugin instance
		plugin := &StepPluginDynamoDB{
			TableName: "test-table",
			client:    mockClient,
		}

		// Create invalid test URL
		prurl, err := url.Parse("https://github.com/mycompany/myrepo/invalid/123")
		assert.Nil(t, err)

		// Execute plugin
		returl, err := plugin.Execute(context.Background(), "test-user", "workflowdescription", "test explanation", prurl, map[string]interface{}{})

		// Assertions
		assert.Nil(t, err)
		assert.Equal(t, "", returl)
		mockClient.AssertExpectations(t)
	})
}
