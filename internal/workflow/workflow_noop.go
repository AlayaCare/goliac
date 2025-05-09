package workflow

import (
	"context"
	"fmt"

	"github.com/goliac-project/goliac/internal/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type NoopImpl struct {
	ws          WorkflowService
	stepPlugins map[string]StepPlugin
}

func NewNoopImpl(ws WorkflowService) Workflow {
	return &NoopImpl{
		ws:          ws,
		stepPlugins: GetPlugins(),
	}
}

func (g *NoopImpl) ExecuteWorkflow(ctx context.Context, repoconfigForceMergeworkflows []string, username, workflowName, explanation string, properties map[string]string, dryrun bool) ([]string, error) {
	var childSpan trace.Span
	if config.Config.OpenTelemetryEnabled {
		ctx, childSpan = otel.Tracer("goliac").Start(ctx, "ExecuteWorkflow")
		defer childSpan.End()
		childSpan.SetAttributes(
			attribute.String("workflow_name", workflowName),
			attribute.String("pr_path_to_merge", properties["pr_url"]),
		)
	}

	// check workflow and acl
	w, err := g.ws.GetWorkflow(ctx, repoconfigForceMergeworkflows, workflowName, "", username)
	if err != nil {
		return nil, fmt.Errorf("unable to execute the workflow: %v", err)
	}

	// execute the workflow
	if dryrun {
		return nil, nil
	}

	responses := []string{}
	for _, step := range w.Spec.Steps {
		plugin := g.stepPlugins[step.Name]
		if plugin == nil {
			return nil, fmt.Errorf("plugin %s not found", step.Name)
		}
		resp, err := plugin.Execute(ctx, username, w.Spec.Description, explanation, nil, step.Properties)
		if err != nil {
			return nil, err
		}
		responses = append(responses, resp)
	}

	return responses, nil
}
