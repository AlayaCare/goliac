package workflow

import (
	"context"
	"fmt"
	"net/url"
	"regexp"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/feature/dynamodb/attributevalue"
	"github.com/aws/aws-sdk-go-v2/service/dynamodb"
	"github.com/goliac-project/goliac/internal/config"
)

// DynamoDBClientInterface defines the interface for DynamoDB operations
type DynamoDBClientInterface interface {
	PutItem(ctx context.Context, params *dynamodb.PutItemInput, optFns ...func(*dynamodb.Options)) (*dynamodb.PutItemOutput, error)
}

type StepPluginDynamoDB struct {
	TableName string
	client    DynamoDBClientInterface
}

// NewStepPluginDynamoDB creates a new DynamoDB plugin with default AWS client
func NewStepPluginDynamoDB() StepPlugin {
	return &StepPluginDynamoDB{
		TableName: config.Config.WorkflowDynamoDBTableName,
	}
}

// NewStepPluginDynamoDBWithClient creates a new DynamoDB plugin with a custom client
// This is used for testing
func NewStepPluginDynamoDBWithClient(client DynamoDBClientInterface) StepPlugin {
	return &StepPluginDynamoDB{
		TableName: config.Config.WorkflowDynamoDBTableName,
		client:    client,
	}
}

type DynamoDBRecord struct {
	Timestamp    string `dynamodbav:"timestamp"`
	PullRequest  string `dynamodbav:"pull_request"`
	GithubCaller string `dynamodbav:"github_caller"`
	Organization string `dynamodbav:"organization"`
	Repository   string `dynamodbav:"repository"`
	PRNumber     string `dynamodbav:"pr_number"`
	Explanation  string `dynamodbav:"explanation"`
}

func (f *StepPluginDynamoDB) Execute(ctx context.Context, username, workflowDescription, explanation string, url *url.URL, properties map[string]interface{}) (string, error) {
	tablename := f.TableName
	if properties["table_name"] != nil {
		tablename = properties["table_name"].(string)
	}

	// Initialize client if not set (for production use)
	if f.client == nil {
		cfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return "", fmt.Errorf("unable to load AWS SDK config: %v", err)
		}
		f.client = dynamodb.NewFromConfig(cfg)
	}

	if url == nil {
		return "", fmt.Errorf("the PR url is not defined (nil)")
	}

	// Extract PR number and repository from URL
	prNumber := ""
	repo := ""
	if url != nil {
		prExtract := regexp.MustCompile(`.*/([^/]*)/pull/(\d+)`)
		prMatch := prExtract.FindStringSubmatch(url.Path)
		if len(prMatch) == 3 {
			repo = prMatch[1]
			prNumber = prMatch[2]
		}
	}

	// Create record
	record := DynamoDBRecord{
		Timestamp:    time.Now().UTC().Format(time.RFC3339),
		PullRequest:  url.String(),
		GithubCaller: username,
		Organization: config.Config.GithubAppOrganization,
		Repository:   repo,
		PRNumber:     prNumber,
		Explanation:  explanation,
	}

	// Convert record to DynamoDB attribute values
	item, err := attributevalue.MarshalMap(record)
	if err != nil {
		return "", fmt.Errorf("failed to marshal record: %v", err)
	}

	// Put item in DynamoDB
	_, err = f.client.PutItem(ctx, &dynamodb.PutItemInput{
		TableName: aws.String(tablename),
		Item:      item,
	})
	if err != nil {
		return "", fmt.Errorf("failed to put item in DynamoDB table %s: %v", tablename, err)
	}

	return "", nil
}
