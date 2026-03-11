package workflows

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

type Document struct {
	FileName      string            `json:"fileName"`
	Path          string            `json:"path"`
	Name          string            `json:"name"`
	TriggerNames  []string          `json:"triggerNames"`
	WorkflowRun   *WorkflowRun      `json:"workflowRun"`
	Uses          []UsesReference   `json:"uses"`
	CheckoutSteps []CheckoutStep    `json:"checkoutSteps"`
	RunSteps      []RunStep         `json:"runSteps"`
	Permissions   []PermissionEntry `json:"permissions"`
}

type WorkflowRun struct {
	Workflows      []string `json:"workflows"`
	Types          []string `json:"types"`
	Branches       []string `json:"branches"`
	BranchesIgnore []string `json:"branchesIgnore"`
}

type UsesReference struct {
	Kind     string `json:"kind"`
	JobID    string `json:"jobId"`
	StepName string `json:"stepName"`
	Uses     string `json:"uses"`
	Line     int    `json:"line"`
}

type CheckoutStep struct {
	JobID              string  `json:"jobId"`
	StepName           string  `json:"stepName"`
	Uses               string  `json:"uses"`
	Line               int     `json:"line"`
	PersistCredentials *string `json:"persistCredentials"`
	Ref                *string `json:"ref"`
	Repository         *string `json:"repository"`
}

type RunStep struct {
	JobID    string `json:"jobId"`
	StepName string `json:"stepName"`
	Run      string `json:"run"`
	Line     int    `json:"line"`
}

type PermissionEntry struct {
	Scope string      `json:"scope"`
	JobID string      `json:"jobId"`
	Line  int         `json:"line"`
	Kind  string      `json:"kind"`
	Value interface{} `json:"value"`
}

func ListFiles(root string) ([]string, error) {
	workflowDir := filepath.Join(normalizeRoot(root), ".github", "workflows")
	entries, err := os.ReadDir(workflowDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []string{}, nil
		}
		return nil, errors.Wrap(err, "read workflow directory")
	}

	files := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(strings.ToLower(name), ".yml") && !strings.HasSuffix(strings.ToLower(name), ".yaml") {
			continue
		}
		files = append(files, name)
	}
	sort.Strings(files)
	return files, nil
}

func ParseAll(root string) ([]Document, error) {
	files, err := ListFiles(root)
	if err != nil {
		return nil, err
	}

	documents := make([]Document, 0, len(files))
	for _, file := range files {
		document, err := ParseFile(root, file)
		if err != nil {
			return nil, err
		}
		documents = append(documents, document)
	}
	return documents, nil
}

func ParseFile(root string, path string) (Document, error) {
	relativePath, absolutePath := resolveWorkflowPath(root, path)
	content, err := os.ReadFile(absolutePath)
	if err != nil {
		return Document{}, errors.Wrapf(err, "read workflow file %s", relativePath)
	}

	var node yaml.Node
	if err := yaml.Unmarshal(content, &node); err != nil {
		return Document{}, errors.Wrapf(err, "parse workflow yaml %s", relativePath)
	}

	document := Document{
		FileName: filepath.Base(relativePath),
		Path:     filepath.ToSlash(relativePath),
	}

	rootNode := unwrapDocument(&node)
	if rootNode == nil {
		return document, nil
	}
	if rootNode.Kind != yaml.MappingNode {
		return Document{}, errors.Errorf("workflow %s must decode to a YAML mapping", relativePath)
	}

	document.Name = scalarValue(mappingValue(rootNode, "name"))
	onNode := mappingValue(rootNode, "on")
	document.TriggerNames = collectTriggerNames(onNode)
	document.WorkflowRun = collectWorkflowRun(onNode)
	document.Permissions = append(document.Permissions, collectPermissions(mappingValue(rootNode, "permissions"), "workflow", "")...)
	document.Uses, document.CheckoutSteps, document.RunSteps = collectJobData(mappingValue(rootNode, "jobs"))

	jobPermissions := collectJobPermissions(mappingValue(rootNode, "jobs"))
	document.Permissions = append(document.Permissions, jobPermissions...)

	return document, nil
}

func normalizeRoot(root string) string {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		return "."
	}
	return trimmed
}

func resolveWorkflowPath(root string, path string) (string, string) {
	trimmed := strings.TrimSpace(path)
	baseRoot := normalizeRoot(root)

	if trimmed == "" {
		relative := filepath.Join(".github", "workflows")
		return filepath.ToSlash(relative), filepath.Join(baseRoot, relative)
	}

	if filepath.IsAbs(trimmed) {
		relative, err := filepath.Rel(baseRoot, trimmed)
		if err != nil {
			return filepath.Base(trimmed), trimmed
		}
		return filepath.ToSlash(relative), trimmed
	}

	normalized := filepath.Clean(trimmed)
	if !strings.Contains(normalized, string(filepath.Separator)) && !strings.Contains(normalized, "/") {
		normalized = filepath.Join(".github", "workflows", normalized)
	}

	return filepath.ToSlash(normalized), filepath.Join(baseRoot, normalized)
}

func unwrapDocument(node *yaml.Node) *yaml.Node {
	if node == nil {
		return nil
	}
	if node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func mappingValue(node *yaml.Node, key string) *yaml.Node {
	node = unwrapDocument(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

func scalarValue(node *yaml.Node) string {
	node = unwrapDocument(node)
	if node == nil {
		return ""
	}
	if node.Kind == yaml.ScalarNode {
		return strings.TrimSpace(node.Value)
	}
	return ""
}

func collectTriggerNames(node *yaml.Node) []string {
	node = unwrapDocument(node)
	if node == nil {
		return []string{}
	}

	switch node.Kind {
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return []string{}
		}
		return []string{value}
	case yaml.SequenceNode:
		ret := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			value := scalarValue(child)
			if value == "" {
				continue
			}
			ret = append(ret, value)
		}
		return ret
	case yaml.MappingNode:
		ret := make([]string, 0, len(node.Content)/2)
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := strings.TrimSpace(node.Content[i].Value)
			if key == "" {
				continue
			}
			ret = append(ret, key)
		}
		return ret
	case yaml.DocumentNode, yaml.AliasNode:
		return []string{}
	default:
		return []string{}
	}
}

func collectWorkflowRun(node *yaml.Node) *WorkflowRun {
	node = unwrapDocument(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}

	workflowRunNode := mappingValue(node, "workflow_run")
	workflowRunNode = unwrapDocument(workflowRunNode)
	if workflowRunNode == nil || workflowRunNode.Kind != yaml.MappingNode {
		return nil
	}

	ret := &WorkflowRun{
		Workflows:      collectScalarList(mappingValue(workflowRunNode, "workflows")),
		Types:          collectScalarList(mappingValue(workflowRunNode, "types")),
		Branches:       collectScalarList(mappingValue(workflowRunNode, "branches")),
		BranchesIgnore: collectScalarList(mappingValue(workflowRunNode, "branches-ignore")),
	}
	return ret
}

func collectScalarList(node *yaml.Node) []string {
	node = unwrapDocument(node)
	if node == nil {
		return []string{}
	}

	switch node.Kind {
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return []string{}
		}
		return []string{value}
	case yaml.SequenceNode:
		ret := make([]string, 0, len(node.Content))
		for _, child := range node.Content {
			value := scalarValue(child)
			if value == "" {
				continue
			}
			ret = append(ret, value)
		}
		return ret
	case yaml.MappingNode, yaml.DocumentNode, yaml.AliasNode:
		return []string{}
	default:
		return []string{}
	}
}

func collectJobData(node *yaml.Node) ([]UsesReference, []CheckoutStep, []RunStep) {
	node = unwrapDocument(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return []UsesReference{}, []CheckoutStep{}, []RunStep{}
	}

	uses := []UsesReference{}
	checkoutSteps := []CheckoutStep{}
	runSteps := []RunStep{}

	for i := 0; i+1 < len(node.Content); i += 2 {
		jobID := strings.TrimSpace(node.Content[i].Value)
		jobNode := unwrapDocument(node.Content[i+1])
		if jobID == "" || jobNode == nil || jobNode.Kind != yaml.MappingNode {
			continue
		}

		if usesNode := mappingValue(jobNode, "uses"); usesNode != nil {
			spec := scalarValue(usesNode)
			if spec != "" {
				uses = append(uses, UsesReference{
					Kind:  "job",
					JobID: jobID,
					Uses:  spec,
					Line:  usesNode.Line,
				})
			}
		}

		stepsNode := mappingValue(jobNode, "steps")
		stepsNode = unwrapDocument(stepsNode)
		if stepsNode == nil || stepsNode.Kind != yaml.SequenceNode {
			continue
		}

		for _, stepNode := range stepsNode.Content {
			stepNode = unwrapDocument(stepNode)
			if stepNode == nil || stepNode.Kind != yaml.MappingNode {
				continue
			}

			stepName := scalarValue(mappingValue(stepNode, "name"))
			runNode := mappingValue(stepNode, "run")
			runValue := scalarValue(runNode)
			if runValue != "" {
				runSteps = append(runSteps, RunStep{
					JobID:    jobID,
					StepName: stepName,
					Run:      runValue,
					Line:     runNode.Line,
				})
			}

			specNode := mappingValue(stepNode, "uses")
			spec := scalarValue(specNode)
			if spec == "" {
				continue
			}

			reference := UsesReference{
				Kind:     "step",
				JobID:    jobID,
				StepName: stepName,
				Uses:     spec,
				Line:     specNode.Line,
			}
			uses = append(uses, reference)

			if !strings.HasPrefix(strings.ToLower(spec), "actions/checkout@") {
				continue
			}

			checkoutSteps = append(checkoutSteps, CheckoutStep{
				JobID:              jobID,
				StepName:           stepName,
				Uses:               spec,
				Line:               specNode.Line,
				PersistCredentials: persistCredentialsValue(stepNode),
				Ref:                stepWithValue(stepNode, "ref"),
				Repository:         stepWithValue(stepNode, "repository"),
			})
		}
	}

	return uses, checkoutSteps, runSteps
}

func persistCredentialsValue(stepNode *yaml.Node) *string {
	return normalizedOptionalString(stepWithValue(stepNode, "persist-credentials"))
}

func stepWithValue(stepNode *yaml.Node, key string) *string {
	withNode := unwrapDocument(mappingValue(stepNode, "with"))
	if withNode == nil || withNode.Kind != yaml.MappingNode {
		return nil
	}
	valueNode := mappingValue(withNode, key)
	value := scalarValue(valueNode)
	if value == "" {
		return nil
	}
	ret := value
	return &ret
}

func normalizedOptionalString(value *string) *string {
	if value == nil {
		return nil
	}
	ret := strings.ToLower(strings.TrimSpace(*value))
	if ret == "" {
		return nil
	}
	return &ret
}

func collectJobPermissions(node *yaml.Node) []PermissionEntry {
	node = unwrapDocument(node)
	if node == nil || node.Kind != yaml.MappingNode {
		return []PermissionEntry{}
	}

	ret := []PermissionEntry{}
	for i := 0; i+1 < len(node.Content); i += 2 {
		jobID := strings.TrimSpace(node.Content[i].Value)
		jobNode := unwrapDocument(node.Content[i+1])
		if jobID == "" || jobNode == nil || jobNode.Kind != yaml.MappingNode {
			continue
		}
		ret = append(ret, collectPermissions(mappingValue(jobNode, "permissions"), "job", jobID)...)
	}
	return ret
}

func collectPermissions(node *yaml.Node, scope string, jobID string) []PermissionEntry {
	node = unwrapDocument(node)
	if node == nil {
		return []PermissionEntry{}
	}

	switch node.Kind {
	case yaml.ScalarNode:
		value := strings.TrimSpace(node.Value)
		if value == "" {
			return []PermissionEntry{}
		}
		return []PermissionEntry{{
			Scope: scope,
			JobID: jobID,
			Line:  node.Line,
			Kind:  "scalar",
			Value: value,
		}}
	case yaml.MappingNode:
		values := map[string]string{}
		for i := 0; i+1 < len(node.Content); i += 2 {
			key := strings.TrimSpace(node.Content[i].Value)
			value := scalarValue(node.Content[i+1])
			if key == "" {
				continue
			}
			values[key] = value
		}
		return []PermissionEntry{{
			Scope: scope,
			JobID: jobID,
			Line:  node.Line,
			Kind:  "map",
			Value: values,
		}}
	case yaml.SequenceNode, yaml.DocumentNode, yaml.AliasNode:
		return []PermissionEntry{{
			Scope: scope,
			JobID: jobID,
			Line:  node.Line,
			Kind:  "unknown",
			Value: nil,
		}}
	default:
		return []PermissionEntry{{
			Scope: scope,
			JobID: jobID,
			Line:  node.Line,
			Kind:  "unknown",
			Value: nil,
		}}
	}
}
