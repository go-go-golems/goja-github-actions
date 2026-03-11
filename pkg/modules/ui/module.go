package uimodule

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/dop251/goja"
	ggjengine "github.com/go-go-golems/go-go-goja/engine"
	"github.com/go-go-golems/go-go-goja/modules"
	gharuntime "github.com/go-go-golems/goja-github-actions/pkg/runtime"
	"github.com/mattn/go-isatty"
	"github.com/pkg/errors"
)

const moduleName = "@goja-gha/ui"

type Dependencies struct {
	Settings *gharuntime.Settings
	Writer   io.Writer
}

type Module struct {
	vm   *goja.Runtime
	deps *Dependencies
}

type reportBuilder struct {
	title       string
	description string
	blocks      []reportBlock
}

type reportBlock interface {
	reportBlock()
}

type statusBlock struct {
	Kind string
	Text string
}

type kvBlock struct {
	Label string
	Value string
}

type listBlock struct {
	Items []string
}

type tableBlock struct {
	Columns []string
	Rows    [][]string
}

type sectionBlock struct {
	Title  string
	Blocks []reportBlock
}

type findingsBlock struct {
	Groups         []findingGroup
	LocationFields []string
}

type findingGroup struct {
	RuleID       string
	Severity     string
	Message      string
	WhyItMatters string
	Remediation  string
	Example      string
	Locations    []findingLocation
}

type findingLocation struct {
	Path string
	Line string
	Hint string
}

type tableOptions struct {
	Columns []string   `json:"columns"`
	Rows    [][]string `json:"rows"`
}

func (statusBlock) reportBlock()   {}
func (kvBlock) reportBlock()       {}
func (listBlock) reportBlock()     {}
func (tableBlock) reportBlock()    {}
func (sectionBlock) reportBlock()  {}
func (findingsBlock) reportBlock() {}

func NewDependencies(settings *gharuntime.Settings) *Dependencies {
	return &Dependencies{
		Settings: settings,
		Writer:   os.Stdout,
	}
}

func Spec(deps *Dependencies) ggjengine.ModuleSpec {
	return ggjengine.NativeModuleSpec{
		ModuleID:   "goja-gha-ui",
		ModuleName: moduleName,
		Loader: func(vm *goja.Runtime, moduleObj *goja.Object) {
			mod := &Module{
				vm:   vm,
				deps: deps,
			}

			exports := moduleObj.Get("exports").(*goja.Object)
			modules.SetExport(exports, moduleName, "report", mod.report)
			modules.SetExport(exports, moduleName, "enabled", mod.enabled)
		},
	}
}

func (m *Module) enabled() bool {
	return !m.deps.Settings.JSONResult
}

func (m *Module) report(title string) goja.Value {
	report := &reportBuilder{
		title: strings.TrimSpace(title),
	}
	return m.newReportObject(report)
}

func (m *Module) newReportObject(report *reportBuilder) *goja.Object {
	reportObject := m.vm.NewObject()

	m.must(reportObject.Set("description", func(text string) *goja.Object {
		report.description = strings.TrimSpace(text)
		return reportObject
	}))
	m.must(reportObject.Set("status", func(call goja.FunctionCall) goja.Value {
		kind := ""
		text := ""
		if len(call.Arguments) > 0 {
			kind = call.Argument(0).String()
		}
		if len(call.Arguments) > 1 {
			text = call.Argument(1).String()
		}
		report.blocks = append(report.blocks, statusBlock{
			Kind: normalizeStatus(kind),
			Text: strings.TrimSpace(text),
		})
		return reportObject
	}))
	m.must(reportObject.Set("success", func(text string) *goja.Object {
		report.blocks = append(report.blocks, statusBlock{Kind: "ok", Text: strings.TrimSpace(text)})
		return reportObject
	}))
	m.must(reportObject.Set("note", func(text string) *goja.Object {
		report.blocks = append(report.blocks, statusBlock{Kind: "info", Text: strings.TrimSpace(text)})
		return reportObject
	}))
	m.must(reportObject.Set("warn", func(text string) *goja.Object {
		report.blocks = append(report.blocks, statusBlock{Kind: "warn", Text: strings.TrimSpace(text)})
		return reportObject
	}))
	m.must(reportObject.Set("error", func(text string) *goja.Object {
		report.blocks = append(report.blocks, statusBlock{Kind: "error", Text: strings.TrimSpace(text)})
		return reportObject
	}))
	m.must(reportObject.Set("kv", func(call goja.FunctionCall) goja.Value {
		label := ""
		if len(call.Arguments) > 0 {
			label = call.Argument(0).String()
		}
		value := goja.Undefined()
		if len(call.Arguments) > 1 {
			value = call.Argument(1)
		}
		report.blocks = append(report.blocks, kvBlock{
			Label: strings.TrimSpace(label),
			Value: stringifyValue(value),
		})
		return reportObject
	}))
	m.must(reportObject.Set("list", func(value goja.Value) *goja.Object {
		report.blocks = append(report.blocks, listBlock{Items: exportStringSlice(value)})
		return reportObject
	}))
	m.must(reportObject.Set("table", func(value goja.Value) *goja.Object {
		options := exportTableOptions(m.vm, value)
		report.blocks = append(report.blocks, tableBlock(options))
		return reportObject
	}))
	m.must(reportObject.Set("section", func(call goja.FunctionCall) goja.Value {
		title := ""
		if len(call.Arguments) > 0 {
			title = call.Argument(0).String()
		}
		callback := goja.Value(goja.Undefined())
		if len(call.Arguments) > 1 {
			callback = call.Argument(1)
		}
		section := &sectionBlock{
			Title: strings.TrimSpace(title),
		}
		report.blocks = append(report.blocks, section)
		sectionObject := m.newSectionObject(&section.Blocks)
		callSectionCallback(m.vm, callback, sectionObject)
		return reportObject
	}))
	m.must(reportObject.Set("render", func() *goja.Object {
		m.must(m.renderReport(report))
		return reportObject
	}))

	return reportObject
}

func (m *Module) newSectionObject(blocks *[]reportBlock) *goja.Object {
	sectionObject := m.vm.NewObject()

	appendBlock := func(block reportBlock) {
		*blocks = append(*blocks, block)
	}

	m.must(sectionObject.Set("status", func(call goja.FunctionCall) goja.Value {
		kind := ""
		text := ""
		if len(call.Arguments) > 0 {
			kind = call.Argument(0).String()
		}
		if len(call.Arguments) > 1 {
			text = call.Argument(1).String()
		}
		appendBlock(statusBlock{Kind: normalizeStatus(kind), Text: strings.TrimSpace(text)})
		return sectionObject
	}))
	m.must(sectionObject.Set("success", func(text string) *goja.Object {
		appendBlock(statusBlock{Kind: "ok", Text: strings.TrimSpace(text)})
		return sectionObject
	}))
	m.must(sectionObject.Set("note", func(text string) *goja.Object {
		appendBlock(statusBlock{Kind: "info", Text: strings.TrimSpace(text)})
		return sectionObject
	}))
	m.must(sectionObject.Set("warn", func(text string) *goja.Object {
		appendBlock(statusBlock{Kind: "warn", Text: strings.TrimSpace(text)})
		return sectionObject
	}))
	m.must(sectionObject.Set("error", func(text string) *goja.Object {
		appendBlock(statusBlock{Kind: "error", Text: strings.TrimSpace(text)})
		return sectionObject
	}))
	m.must(sectionObject.Set("kv", func(call goja.FunctionCall) goja.Value {
		label := ""
		if len(call.Arguments) > 0 {
			label = call.Argument(0).String()
		}
		value := goja.Undefined()
		if len(call.Arguments) > 1 {
			value = call.Argument(1)
		}
		appendBlock(kvBlock{Label: strings.TrimSpace(label), Value: stringifyValue(value)})
		return sectionObject
	}))
	m.must(sectionObject.Set("list", func(value goja.Value) *goja.Object {
		appendBlock(listBlock{Items: exportStringSlice(value)})
		return sectionObject
	}))
	m.must(sectionObject.Set("table", func(value goja.Value) *goja.Object {
		appendBlock(tableBlock(exportTableOptions(m.vm, value)))
		return sectionObject
	}))
	m.must(sectionObject.Set("section", func(call goja.FunctionCall) goja.Value {
		title := ""
		if len(call.Arguments) > 0 {
			title = call.Argument(0).String()
		}
		callback := goja.Value(goja.Undefined())
		if len(call.Arguments) > 1 {
			callback = call.Argument(1)
		}
		section := &sectionBlock{Title: strings.TrimSpace(title)}
		appendBlock(section)
		nestedObject := m.newSectionObject(&section.Blocks)
		callSectionCallback(m.vm, callback, nestedObject)
		return sectionObject
	}))
	m.must(sectionObject.Set("findings", func(call goja.FunctionCall) goja.Value {
		fb := exportFindingsBlock(m.vm, call)
		appendBlock(fb)
		return sectionObject
	}))

	return sectionObject
}

func (m *Module) renderReport(report *reportBuilder) error {
	if report == nil || m.deps == nil || m.deps.Settings == nil {
		return nil
	}
	if m.deps.Settings.JSONResult {
		return nil
	}

	writer := m.deps.Writer
	if writer == nil {
		writer = os.Stdout
	}

	rendered := renderTextReport(report, supportsColor(writer))
	if strings.TrimSpace(rendered) == "" {
		return nil
	}

	if _, err := io.WriteString(writer, rendered); err != nil {
		return errors.Wrap(err, "write report")
	}
	if m.deps.Settings.State == nil {
		m.deps.Settings.State = &gharuntime.State{
			Environment: m.deps.Settings.ProcessEnv(),
		}
	}
	m.deps.Settings.State.HumanOutputRendered = true
	return nil
}

func renderTextReport(report *reportBuilder, color bool) string {
	var buffer bytes.Buffer

	if strings.TrimSpace(report.title) != "" {
		writeLine(&buffer, styleHeading(report.title, color))
		writeLine(&buffer, strings.Repeat("=", len(report.title)))
		writeLine(&buffer, "")
	}

	if report.description != "" {
		for _, line := range wordWrap(report.description, 70) {
			writeLine(&buffer, "  "+line)
		}
		writeLine(&buffer, "")
	}

	renderBlocks(&buffer, report.blocks, color)

	return strings.TrimRight(buffer.String(), "\n") + "\n"
}

func renderBlocks(buffer *bytes.Buffer, blocks []reportBlock, color bool) {
	if len(blocks) == 0 {
		return
	}

	kvWidth := keyValueWidth(blocks)
	for i, block := range blocks {
		switch typed := block.(type) {
		case statusBlock:
			writeLine(buffer, fmt.Sprintf("%s  %s", styleStatusLabel(typed.Kind, color), typed.Text))
		case kvBlock:
			label := typed.Label
			if label == "" {
				label = "Value"
			}
			writeLine(buffer, fmt.Sprintf("  %-*s  %s", kvWidth, label, typed.Value))
		case listBlock:
			for _, item := range typed.Items {
				writeLine(buffer, "  - "+item)
			}
		case tableBlock:
			renderTable(buffer, typed, color)
		case *sectionBlock:
			if typed.Title != "" {
				writeLine(buffer, styleSectionHeading(typed.Title, color))
				writeLine(buffer, strings.Repeat("-", len(typed.Title)))
			}
			if len(typed.Blocks) > 0 {
				renderBlocks(buffer, typed.Blocks, color)
			}
		case findingsBlock:
			renderFindings(buffer, typed, color)
		}
		if i != len(blocks)-1 && !sameBlockType(block, blocks[i+1]) {
			writeLine(buffer, "")
		}
	}
}

func sameBlockType(a, b reportBlock) bool {
	switch a.(type) {
	case kvBlock:
		_, ok := b.(kvBlock)
		return ok
	case statusBlock:
		_, ok := b.(statusBlock)
		return ok
	default:
		return false
	}
}

func renderTable(buffer *bytes.Buffer, table tableBlock, color bool) {
	if len(table.Columns) == 0 {
		return
	}

	widths := make([]int, len(table.Columns))
	for i, column := range table.Columns {
		widths[i] = len(column)
	}
	for _, row := range table.Rows {
		for i, cell := range row {
			if i >= len(widths) {
				break
			}
			if len(cell) > widths[i] {
				widths[i] = len(cell)
			}
		}
	}

	writeLine(buffer, formatRow(table.Columns, widths, color))
	writeLine(buffer, formatDivider(widths))
	for _, row := range table.Rows {
		cells := make([]string, len(table.Columns))
		copy(cells, row)
		writeLine(buffer, formatRow(cells, widths, false))
	}
}

func formatRow(cells []string, widths []int, color bool) string {
	parts := make([]string, 0, len(widths))
	for i, width := range widths {
		value := ""
		if i < len(cells) {
			value = cells[i]
		}
		if color {
			value = ansi("1", value)
		}
		parts = append(parts, fmt.Sprintf("%-*s", width, value))
	}
	return strings.Join(parts, "  ")
}

func formatDivider(widths []int) string {
	parts := make([]string, 0, len(widths))
	for _, width := range widths {
		parts = append(parts, strings.Repeat("-", width))
	}
	return strings.Join(parts, "  ")
}

func keyValueWidth(blocks []reportBlock) int {
	width := 12
	for _, block := range blocks {
		if kv, ok := block.(kvBlock); ok {
			if len(kv.Label) > width {
				width = len(kv.Label)
			}
		}
	}
	if width > 28 {
		return 28
	}
	return width
}

func styleHeading(text string, color bool) string {
	if !color {
		return text
	}
	return ansi("1", text)
}

func styleSectionHeading(text string, color bool) string {
	if !color {
		return text
	}
	return ansi("1;36", text)
}

func styleStatusLabel(kind string, color bool) string {
	normalized := normalizeStatus(kind)
	var label string
	switch normalized {
	case "ok":
		label = "[ OK ]"
	case "warn":
		label = "[WARN]"
	case "error":
		label = "[ERR ]"
	case "skip":
		label = "[SKIP]"
	default:
		label = "[INFO]"
	}
	if !color {
		return label
	}

	switch normalized {
	case "ok":
		return ansi("1;32", label)
	case "warn":
		return ansi("1;33", label)
	case "error":
		return ansi("1;31", label)
	case "skip":
		return ansi("1;35", label)
	default:
		return ansi("1;34", label)
	}
}

func ansi(code string, text string) string {
	return "\x1b[" + code + "m" + text + "\x1b[0m"
}

func writeLine(buffer *bytes.Buffer, line string) {
	buffer.WriteString(line)
	buffer.WriteByte('\n')
}

func supportsColor(writer io.Writer) bool {
	file, ok := writer.(*os.File)
	if !ok {
		return false
	}
	if os.Getenv("NO_COLOR") != "" || strings.EqualFold(os.Getenv("TERM"), "dumb") {
		return false
	}
	return isatty.IsTerminal(file.Fd()) || isatty.IsCygwinTerminal(file.Fd())
}

func normalizeStatus(kind string) string {
	switch strings.ToLower(strings.TrimSpace(kind)) {
	case "ok", "success":
		return "ok"
	case "warn", "warning":
		return "warn"
	case "error", "err":
		return "error"
	case "skip", "skipped":
		return "skip"
	default:
		return "info"
	}
}

func stringifyValue(value goja.Value) string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return ""
	}

	exported := value.Export()
	switch typed := exported.(type) {
	case string:
		return typed
	case int64:
		return strconv.FormatInt(typed, 10)
	case int:
		return strconv.Itoa(typed)
	case float64:
		return strconv.FormatFloat(typed, 'f', -1, 64)
	case bool:
		if typed {
			return "true"
		}
		return "false"
	default:
		return fmt.Sprint(exported)
	}
}

func exportStringSlice(value goja.Value) []string {
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return nil
	}

	exported := value.Export()
	switch typed := exported.(type) {
	case []string:
		return typed
	case []interface{}:
		items := make([]string, 0, len(typed))
		for _, item := range typed {
			items = append(items, fmt.Sprint(item))
		}
		return items
	default:
		return []string{fmt.Sprint(exported)}
	}
}

func exportTableOptions(vm *goja.Runtime, value goja.Value) tableOptions {
	options := tableOptions{}
	if value == nil || goja.IsUndefined(value) || goja.IsNull(value) {
		return options
	}

	object := value.ToObject(vm)
	if object == nil {
		return options
	}
	options.Columns = exportStringSlice(object.Get("columns"))
	rowsValue := object.Get("rows")
	if rowsValue == nil || goja.IsUndefined(rowsValue) || goja.IsNull(rowsValue) {
		return options
	}
	rowsObject := rowsValue.ToObject(vm)
	if rowsObject == nil {
		return options
	}
	for _, key := range rowsObject.Keys() {
		options.Rows = append(options.Rows, exportStringSlice(rowsObject.Get(key)))
	}
	return options
}

func callSectionCallback(vm *goja.Runtime, callback goja.Value, sectionObject *goja.Object) {
	callable, ok := goja.AssertFunction(callback)
	if !ok {
		panic(vm.NewTypeError("section expects a function"))
	}
	if _, err := callable(goja.Undefined(), sectionObject); err != nil {
		panic(err)
	}
}

func exportFindingsBlock(vm *goja.Runtime, call goja.FunctionCall) findingsBlock {
	fb := findingsBlock{}
	if len(call.Arguments) == 0 {
		return fb
	}

	findingsValue := call.Argument(0)
	if findingsValue == nil || goja.IsUndefined(findingsValue) || goja.IsNull(findingsValue) {
		return fb
	}

	// Parse options (second argument)
	locationFields := []string{"path", "line", "uses"}
	if len(call.Arguments) > 1 {
		optsValue := call.Argument(1)
		if optsValue != nil && !goja.IsUndefined(optsValue) && !goja.IsNull(optsValue) {
			optsObj := optsValue.ToObject(vm)
			if optsObj != nil {
				if lf := optsObj.Get("locationFields"); lf != nil && !goja.IsUndefined(lf) {
					locationFields = exportStringSlice(lf)
				}
			}
		}
	}
	fb.LocationFields = locationFields

	// Extract findings array into groups keyed by ruleId
	findingsObj := findingsValue.ToObject(vm)
	if findingsObj == nil {
		return fb
	}

	type rawFinding struct {
		ruleID       string
		severity     string
		message      string
		whyItMatters string
		remediation  string
		example      string
		evidence     map[string]string
	}

	groupOrder := []string{}
	groupMap := map[string][]rawFinding{}

	for _, key := range findingsObj.Keys() {
		item := findingsObj.Get(key)
		if item == nil || goja.IsUndefined(item) || goja.IsNull(item) {
			continue
		}
		obj := item.ToObject(vm)
		if obj == nil {
			continue
		}

		rf := rawFinding{
			ruleID:       getStringField(obj, "ruleId"),
			severity:     getStringField(obj, "severity"),
			message:      getStringField(obj, "message"),
			whyItMatters: getStringField(obj, "whyItMatters"),
			evidence:     map[string]string{},
		}

		// Extract remediation
		remValue := obj.Get("remediation")
		if remValue != nil && !goja.IsUndefined(remValue) && !goja.IsNull(remValue) {
			remObj := remValue.ToObject(vm)
			if remObj != nil {
				rf.remediation = getStringField(remObj, "summary")
				rf.example = getStringField(remObj, "example")
			}
		}

		// Extract evidence fields
		evValue := obj.Get("evidence")
		if evValue != nil && !goja.IsUndefined(evValue) && !goja.IsNull(evValue) {
			evObj := evValue.ToObject(vm)
			if evObj != nil {
				for _, field := range locationFields {
					val := evObj.Get(field)
					if val != nil && !goja.IsUndefined(val) && !goja.IsNull(val) {
						rf.evidence[field] = stringifyValue(val)
					}
				}
			}
		}

		if _, exists := groupMap[rf.ruleID]; !exists {
			groupOrder = append(groupOrder, rf.ruleID)
		}
		groupMap[rf.ruleID] = append(groupMap[rf.ruleID], rf)
	}

	// Build finding groups in order
	for _, ruleID := range groupOrder {
		findings := groupMap[ruleID]
		if len(findings) == 0 {
			continue
		}
		first := findings[0]
		group := findingGroup{
			RuleID:       ruleID,
			Severity:     strings.ToUpper(first.severity),
			Message:      first.message,
			WhyItMatters: first.whyItMatters,
			Remediation:  first.remediation,
			Example:      first.example,
		}

		for _, f := range findings {
			loc := findingLocation{}
			if v, ok := f.evidence["path"]; ok {
				loc.Path = v
			}
			if v, ok := f.evidence["line"]; ok {
				loc.Line = v
			}
			// Build hint from remaining location fields (excluding path and line)
			var hintParts []string
			for _, field := range locationFields {
				if field == "path" || field == "line" {
					continue
				}
				if v, ok := f.evidence[field]; ok && v != "" {
					hintParts = append(hintParts, v)
				}
			}
			if len(hintParts) > 0 {
				loc.Hint = strings.Join(hintParts, "  ")
			}
			group.Locations = append(group.Locations, loc)
		}

		fb.Groups = append(fb.Groups, group)
	}

	return fb
}

func getStringField(obj *goja.Object, field string) string {
	val := obj.Get(field)
	if val == nil || goja.IsUndefined(val) || goja.IsNull(val) {
		return ""
	}
	return strings.TrimSpace(val.String())
}

func renderFindings(buffer *bytes.Buffer, fb findingsBlock, color bool) {
	for i, group := range fb.Groups {
		// Header: ruleId + count x SEVERITY
		countLabel := fmt.Sprintf("%d x %s", len(group.Locations), group.Severity)
		if color {
			countLabel = fmt.Sprintf("%d x %s", len(group.Locations), styleSeverity(group.Severity, color))
		}
		header := group.RuleID
		if color {
			header = ansi("1", header)
		}
		// Right-align count on ~72-char line
		padding := 72 - len(group.RuleID) - len(fmt.Sprintf("%d x %s", len(group.Locations), group.Severity))
		if padding < 2 {
			padding = 2
		}
		writeLine(buffer, header+strings.Repeat(" ", padding)+countLabel)

		// Message
		for _, line := range wordWrap(group.Message, 68) {
			writeLine(buffer, "  "+line)
		}

		// Why it matters
		if group.WhyItMatters != "" {
			writeLine(buffer, "")
			label := "Why it matters"
			if color {
				label = ansi("2", label)
			}
			writeLine(buffer, "  "+label)
			for _, line := range wordWrap(group.WhyItMatters, 66) {
				writeLine(buffer, "    "+line)
			}
		}

		// Remediation
		if group.Remediation != "" {
			writeLine(buffer, "")
			label := "Remediation"
			if color {
				label = ansi("2", label)
			}
			writeLine(buffer, "  "+label)
			for _, line := range wordWrap(group.Remediation, 66) {
				writeLine(buffer, "    "+line)
			}
			if group.Example != "" {
				prefix := "    Example: "
				wrapped := wordWrap(group.Example, 66-len("Example: "))
				if len(wrapped) > 0 {
					writeLine(buffer, prefix+wrapped[0])
					for _, line := range wrapped[1:] {
						writeLine(buffer, "    "+strings.Repeat(" ", len("Example: "))+line)
					}
				}
			}
		}

		// Locations grouped by path
		if len(group.Locations) > 0 && hasLocationDetail(group.Locations) {
			writeLine(buffer, "")
			label := "Locations"
			if color {
				label = ansi("2", label)
			}
			writeLine(buffer, "  "+label)
			renderGroupedLocations(buffer, group.Locations)
		}

		if i < len(fb.Groups)-1 {
			writeLine(buffer, "")
		}
	}
}

func hasLocationDetail(locs []findingLocation) bool {
	for _, loc := range locs {
		if loc.Path != "" || loc.Line != "" || loc.Hint != "" {
			return true
		}
	}
	return false
}

func renderGroupedLocations(buffer *bytes.Buffer, locations []findingLocation) {
	// Group by path
	type pathGroup struct {
		path      string
		locations []findingLocation
	}
	var groups []pathGroup
	groupIndex := map[string]int{}

	for _, loc := range locations {
		path := loc.Path
		if path == "" {
			path = "(no path)"
		}
		idx, exists := groupIndex[path]
		if !exists {
			idx = len(groups)
			groups = append(groups, pathGroup{path: path})
			groupIndex[path] = idx
		}
		groups[idx].locations = append(groups[idx].locations, loc)
	}

	for _, pg := range groups {
		if len(groups) > 1 || pg.path != "(no path)" {
			writeLine(buffer, "    "+pg.path)
		}
		for _, loc := range pg.locations {
			line := loc.Line
			if line == "" {
				line = "-"
			}
			hint := ""
			if loc.Hint != "" {
				hint = "  " + loc.Hint
			}
			if len(groups) > 1 || pg.path != "(no path)" {
				writeLine(buffer, fmt.Sprintf("       :%s%s", line, hint))
			} else {
				writeLine(buffer, fmt.Sprintf("    :%s%s", line, hint))
			}
		}
	}
}

func styleSeverity(severity string, color bool) string {
	if !color {
		return severity
	}
	switch strings.ToUpper(severity) {
	case "CRITICAL":
		return ansi("1;31", severity)
	case "HIGH":
		return ansi("1;31", severity)
	case "MEDIUM":
		return ansi("1;33", severity)
	case "LOW":
		return ansi("1;34", severity)
	default:
		return ansi("2", severity)
	}
}

func wordWrap(text string, width int) []string {
	if width <= 0 {
		width = 70
	}
	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	current := words[0]
	for _, word := range words[1:] {
		if len(current)+1+len(word) > width {
			lines = append(lines, current)
			current = word
		} else {
			current += " " + word
		}
	}
	lines = append(lines, current)
	return lines
}

func (m *Module) must(err error) {
	if err != nil {
		panic(m.vm.NewGoError(err))
	}
}
