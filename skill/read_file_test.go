package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestMain 在所有测试运行前切换工作目录到项目根目录
// 因为 read_file.go 中使用了相对路径 "skill/bash/read_lines.sh"
// go test 默认在 skill/ 目录下运行，需要切换到上级目录
func TestMain(m *testing.M) {
	// 保存原始工作目录
	origDir, _ := os.Getwd()
	// 切换到项目根目录（skill 的上级目录）
	os.Chdir(filepath.Join(origDir, ".."))
	code := m.Run()
	// 恢复原始工作目录
	os.Chdir(origDir)
	os.Exit(code)
}

// ============================================================
// 辅助工具函数
// ============================================================

// createTempFile 创建临时测试文件，返回文件路径和清理函数
func createTempFile(t *testing.T, content string) (string, func()) {
	t.Helper()
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "test_read_file.txt")
	err := os.WriteFile(tmpFile, []byte(content), 0644)
	if err != nil {
		t.Fatalf("创建临时文件失败: %v", err)
	}
	return tmpFile, func() {
		os.Remove(tmpFile)
	}
}

// buildArgsJSON 构建 Execute 方法的 JSON 参数
func buildArgsJSON(path string, start int, lineCount int) string {
	args := map[string]interface{}{
		"path": path,
	}
	if start > 0 {
		args["start"] = start
	}
	if lineCount > 0 {
		args["line_count"] = lineCount
	}
	data, _ := json.Marshal(args)
	return string(data)
}

// ============================================================
// 一、Skill 接口实现测试
// ============================================================

// TC-001: ReadFileSkill 实现了 Skill 接口
func TestReadFileSkill_ImplementsSkillInterface(t *testing.T) {
	var _ Skill = &ReadFileSkill{}
	t.Log("ReadFileSkill 正确实现了 Skill 接口")
}

// TC-002: Name() 返回正确的工具名称
func TestReadFileSkill_Name(t *testing.T) {
	rf := &ReadFileSkill{}
	name := rf.Name()
	if name != "read_file" {
		t.Errorf("Name() 应返回 'read_file'，实际返回 '%s'", name)
	}
}

// TC-003: Description() 返回正确的工具定义
func TestReadFileSkill_Description(t *testing.T) {
	rf := &ReadFileSkill{}
	desc := rf.Description()

	// 验证 Type
	if desc.Type != "function" {
		t.Errorf("Description().Type 应为 'function'，实际为 '%s'", desc.Type)
	}

	// 验证 Function.Name
	if desc.Function.Name != "read_file" {
		t.Errorf("Function.Name 应为 'read_file'，实际为 '%s'", desc.Function.Name)
	}

	// 验证 Function.Description 非空
	if desc.Function.Description == "" {
		t.Error("Function.Description 不应为空")
	}

	// 验证 Parameters 结构
	params := desc.Function.Parameters
	if params == nil {
		t.Fatal("Parameters 不应为 nil")
	}

	// 验证 type 为 object
	if params["type"] != "object" {
		t.Errorf("Parameters.type 应为 'object'，实际为 '%v'", params["type"])
	}

	// 验证 properties 包含 path、start、line_count
	properties, ok := params["properties"].(map[string]interface{})
	if !ok {
		t.Fatal("Parameters.properties 应为 map[string]interface{}")
	}

	requiredProps := []string{"path", "start", "line_count"}
	for _, prop := range requiredProps {
		if _, exists := properties[prop]; !exists {
			t.Errorf("Parameters.properties 应包含 '%s'", prop)
		}
	}

	// 验证 required 包含 path
	required, ok := params["required"].([]string)
	if !ok {
		t.Fatal("Parameters.required 应为 []string")
	}
	hasPath := false
	for _, r := range required {
		if r == "path" {
			hasPath = true
			break
		}
	}
	if !hasPath {
		t.Error("Parameters.required 应包含 'path'")
	}
}

// TC-004: Description() 中每个 property 都有 type 和 description
func TestReadFileSkill_Description_PropertyDetails(t *testing.T) {
	rf := &ReadFileSkill{}
	desc := rf.Description()
	properties := desc.Function.Parameters["properties"].(map[string]interface{})

	for propName, propVal := range properties {
		propMap, ok := propVal.(map[string]interface{})
		if !ok {
			t.Errorf("property '%s' 应为 map[string]interface{}", propName)
			continue
		}
		if _, hasType := propMap["type"]; !hasType {
			t.Errorf("property '%s' 缺少 'type' 字段", propName)
		}
		if _, hasDesc := propMap["description"]; !hasDesc {
			t.Errorf("property '%s' 缺少 'description' 字段", propName)
		}
	}
}

// ============================================================
// 二、AllSkills 注册测试
// ============================================================

// TC-005: ReadFileSkill 已注册到 AllSkills 中
func TestAllSkills_ContainsReadFileSkill(t *testing.T) {
	skills := AllSkills()
	found := false
	for _, s := range skills {
		if s.Name() == "read_file" {
			found = true
			break
		}
	}
	if !found {
		t.Error("AllSkills() 中应包含 ReadFileSkill")
	}
}

// TC-006: AllSkills 中的工具名称不重复
func TestAllSkills_NoDuplicateNames(t *testing.T) {
	skills := AllSkills()
	nameSet := make(map[string]bool)
	for _, s := range skills {
		name := s.Name()
		if nameSet[name] {
			t.Errorf("AllSkills() 中存在重复的工具名称: '%s'", name)
		}
		nameSet[name] = true
	}
}

// ============================================================
// 三、Execute 参数解析测试
// ============================================================

// TC-007: 无效 JSON 参数应返回错误信息
func TestReadFileSkill_Execute_InvalidJSON(t *testing.T) {
	rf := &ReadFileSkill{}
	result := rf.Execute("invalid json")
	if !strings.Contains(result, "解析工具参数失败") {
		t.Errorf("无效 JSON 应返回解析失败信息，实际返回: %s", result)
	}
}

// TC-008: 空路径应返回错误信息
func TestReadFileSkill_Execute_EmptyPath(t *testing.T) {
	rf := &ReadFileSkill{}
	result := rf.Execute(`{"path": ""}`)
	if !strings.Contains(result, "文件路径不能为空") {
		t.Errorf("空路径应返回'文件路径不能为空'，实际返回: %s", result)
	}
}

// TC-009: 不存在的文件应返回错误信息
func TestReadFileSkill_Execute_FileNotExist(t *testing.T) {
	rf := &ReadFileSkill{}
	result := rf.Execute(`{"path": "/tmp/nonexistent_file_12345.txt"}`)
	if !strings.Contains(result, "文件不存在") {
		t.Errorf("不存在的文件应返回'文件不存在'，实际返回: %s", result)
	}
}

// TC-010: 缺少 path 参数（JSON 中无 path 字段）
func TestReadFileSkill_Execute_MissingPath(t *testing.T) {
	rf := &ReadFileSkill{}
	result := rf.Execute(`{"start": 1}`)
	if !strings.Contains(result, "文件路径不能为空") {
		t.Errorf("缺少 path 参数应返回'文件路径不能为空'，实际返回: %s", result)
	}
}

// ============================================================
// 四、Execute 正常读取测试
// ============================================================

// TC-011: 读取整个文件（传 start=1）
func TestReadFileSkill_Execute_ReadEntireFile(t *testing.T) {
	content := "第一行\n第二行\n第三行\n第四行\n第五行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	// 使用 start=1, line_count=0 来读取整个文件（绕过 start=0 的 bug）
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	if !strings.Contains(result, "第一行") {
		t.Errorf("应包含'第一行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第五行") {
		t.Errorf("应包含'第五行'，实际返回: %s", result)
	}
}

// TC-011b: 仅传 path 时（start 和 line_count 未传），应正确读取整个文件
// 之前存在 Bug：start 默认为 0 导致读取为空，现已修复（添加了 if args.Start <= 0 { args.Start = 1 }）
func TestReadFileSkill_Execute_ReadEntireFile_DefaultStart(t *testing.T) {
	content := "第一行\n第二行\n第三行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	// 不传 start 和 line_count，Go 默认值为 0，Execute() 中会修正为 1
	argsJSON := `{"path": "` + tmpFile + `"}`
	result := rf.Execute(argsJSON)

	// 修复后应正确读取整个文件
	if !strings.Contains(result, "第一行") {
		t.Errorf("仅传 path 时应包含'第一行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第三行") {
		t.Errorf("仅传 path 时应包含'第三行'，实际返回: %s", result)
	}
	if result == "" {
		t.Error("仅传 path 时不应返回空内容（Bug-1 应已修复）")
	}
}

// TC-012: 从指定行开始读取到末尾
func TestReadFileSkill_Execute_ReadFromStart(t *testing.T) {
	content := "第一行\n第二行\n第三行\n第四行\n第五行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 3, 0)
	result := rf.Execute(argsJSON)

	if strings.Contains(result, "第一行") {
		t.Errorf("从第3行开始读取不应包含'第一行'，实际返回: %s", result)
	}
	if strings.Contains(result, "第二行") {
		t.Errorf("从第3行开始读取不应包含'第二行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第三行") {
		t.Errorf("从第3行开始读取应包含'第三行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第五行") {
		t.Errorf("从第3行开始读取应包含'第五行'，实际返回: %s", result)
	}
}

// TC-013: 从指定行开始读取指定行数
func TestReadFileSkill_Execute_ReadRange(t *testing.T) {
	content := "第一行\n第二行\n第三行\n第四行\n第五行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 2, 2)
	result := rf.Execute(argsJSON)

	if strings.Contains(result, "第一行") {
		t.Errorf("读取第2-3行不应包含'第一行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第二行") {
		t.Errorf("读取第2-3行应包含'第二行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第三行") {
		t.Errorf("读取第2-3行应包含'第三行'，实际返回: %s", result)
	}
	if strings.Contains(result, "第四行") {
		t.Errorf("读取第2-3行不应包含'第四行'，实际返回: %s", result)
	}
}

// TC-014: 读取单行文件
func TestReadFileSkill_Execute_SingleLineFile(t *testing.T) {
	content := "唯一的一行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	if !strings.Contains(result, "唯一的一行") {
		t.Errorf("应包含'唯一的一行'，实际返回: %s", result)
	}
}

// TC-015: 读取空文件
func TestReadFileSkill_Execute_EmptyFile(t *testing.T) {
	tmpFile, cleanup := createTempFile(t, "")
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 0, 0)
	result := rf.Execute(argsJSON)

	// 空文件读取不应报错
	if strings.Contains(result, "失败") || strings.Contains(result, "错误") {
		t.Errorf("读取空文件不应报错，实际返回: %s", result)
	}
}

// ============================================================
// 五、Execute 边界情况测试
// ============================================================

// TC-016: 起始行超过文件总行数
func TestReadFileSkill_Execute_StartBeyondFileEnd(t *testing.T) {
	content := "第一行\n第二行\n第三行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 100, 0)
	result := rf.Execute(argsJSON)

	// 起始行超出范围，应返回空内容或警告，不应报错
	if strings.Contains(result, "失败") {
		t.Errorf("起始行超出范围不应报执行失败，实际返回: %s", result)
	}
}

// TC-017: 读取行数超过剩余行数
func TestReadFileSkill_Execute_LineCountBeyondRemaining(t *testing.T) {
	content := "第一行\n第二行\n第三行\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 2, 100)
	result := rf.Execute(argsJSON)

	// 应读取到文件末尾，不应报错
	if !strings.Contains(result, "第二行") {
		t.Errorf("应包含'第二行'，实际返回: %s", result)
	}
	if !strings.Contains(result, "第三行") {
		t.Errorf("应包含'第三行'，实际返回: %s", result)
	}
}

// TC-018: 读取包含特殊字符的文件
func TestReadFileSkill_Execute_SpecialCharacters(t *testing.T) {
	content := "包含特殊字符: $HOME `echo hello` \"引号\" '单引号'\n第二行: tab\there\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	if !strings.Contains(result, "包含特殊字符") {
		t.Errorf("应正确读取包含特殊字符的文件，实际返回: %s", result)
	}
}

// TC-019: 读取包含中文的文件
func TestReadFileSkill_Execute_ChineseContent(t *testing.T) {
	content := "你好世界\n这是中文测试\n第三行中文内容\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	if !strings.Contains(result, "你好世界") {
		t.Errorf("应正确读取中文内容，实际返回: %s", result)
	}
	if !strings.Contains(result, "第三行中文内容") {
		t.Errorf("应包含'第三行中文内容'，实际返回: %s", result)
	}
}

// TC-020: 读取大文件（100行）
func TestReadFileSkill_Execute_LargeFile(t *testing.T) {
	var sb strings.Builder
	for i := 1; i <= 100; i++ {
		sb.WriteString("这是第" + strings.Repeat("x", 10) + "行内容\n")
	}
	tmpFile, cleanup := createTempFile(t, sb.String())
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	if result == "" {
		t.Error("读取大文件不应返回空结果")
	}
	if strings.Contains(result, "失败") {
		t.Errorf("读取大文件不应报错，实际返回: %s", result)
	}
}

// ============================================================
// 六、bash 脚本语法验证测试
// ============================================================

// TC-021: bash 脚本语法检查
func TestReadLinesScript_SyntaxCheck(t *testing.T) {
	// 通过 bash -n 检查脚本语法
	scriptPath := "bash/read_lines.sh"
	if _, err := os.Stat(scriptPath); os.IsNotExist(err) {
		t.Skip("脚本文件不存在，跳过语法检查")
	}

	// 使用 Execute 读取一个已知存在的文件来间接验证脚本可执行
	tmpFile, cleanup := createTempFile(t, "语法测试\n")
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 0, 0)
	result := rf.Execute(argsJSON)

	if strings.Contains(result, "执行脚本失败") {
		t.Errorf("bash 脚本执行失败，可能存在语法错误: %s", result)
	}
}

// ============================================================
// 七、工具执行结果格式测试
// ============================================================

// TC-022: 执行结果为字符串类型（符合 Skill 接口规范）
func TestReadFileSkill_Execute_ReturnsString(t *testing.T) {
	content := "测试内容\n"
	tmpFile, cleanup := createTempFile(t, content)
	defer cleanup()

	rf := &ReadFileSkill{}
	argsJSON := buildArgsJSON(tmpFile, 1, 0)
	result := rf.Execute(argsJSON)

	// 验证返回值是非空字符串
	if result == "" {
		t.Error("Execute() 不应返回空字符串")
	}
}

// TC-023: 错误信息格式统一（包含中文描述）
func TestReadFileSkill_Execute_ErrorMessageFormat(t *testing.T) {
	rf := &ReadFileSkill{}

	// 测试各种错误场景的信息格式
	testCases := []struct {
		name     string
		argsJSON string
		expected string
	}{
		{"无效JSON", "invalid", "解析工具参数失败"},
		{"空路径", `{"path": ""}`, "文件路径不能为空"},
		{"文件不存在", `{"path": "/tmp/nonexistent_12345.txt"}`, "文件不存在"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := rf.Execute(tc.argsJSON)
			if !strings.Contains(result, tc.expected) {
				t.Errorf("错误信息应包含 '%s'，实际返回: %s", tc.expected, result)
			}
		})
	}
}

// ============================================================
// 八、Agent 集成测试
// ============================================================

// TC-024: Agent.ToolByName 中可以找到 read_file
func TestAgent_ToolByName_ContainsReadFile(t *testing.T) {
	skills := AllSkills()
	toolByName := make(map[string]Skill)
	for _, s := range skills {
		toolByName[s.Name()] = s
	}

	tool, ok := toolByName["read_file"]
	if !ok {
		t.Fatal("ToolByName 中应包含 'read_file'")
	}
	if tool.Name() != "read_file" {
		t.Errorf("工具名称应为 'read_file'，实际为 '%s'", tool.Name())
	}
}

// TC-025: 新增工具无需修改 Agent 代码（验证注册机制）
func TestSkillRegistration_NoAgentCodeChange(t *testing.T) {
	// 验证 AllSkills() 返回的所有工具都实现了 Skill 接口
	// 新增工具只需：1. 实现 Skill 接口 2. 在 AllSkills() 中注册
	skills := AllSkills()
	for _, s := range skills {
		// 验证每个 skill 的 Name() 非空
		if s.Name() == "" {
			t.Error("Skill.Name() 不应返回空字符串")
		}
		// 验证每个 skill 的 Description() 有效
		desc := s.Description()
		if desc.Type == "" {
			t.Errorf("Skill '%s' 的 Description().Type 不应为空", s.Name())
		}
		if desc.Function.Name == "" {
			t.Errorf("Skill '%s' 的 Description().Function.Name 不应为空", s.Name())
		}
	}
	t.Logf("AllSkills() 返回 %d 个工具，全部正确实现 Skill 接口", len(skills))
}

// TC-026: 验证 read_file 的 Description 可以被正确序列化为 JSON（用于 API 请求）
func TestReadFileSkill_Description_JSONSerializable(t *testing.T) {
	rf := &ReadFileSkill{}
	desc := rf.Description()

	data, err := json.Marshal(desc)
	if err != nil {
		t.Fatalf("Description() 序列化为 JSON 失败: %v", err)
	}

	if len(data) == 0 {
		t.Error("序列化后的 JSON 不应为空")
	}

	// 验证 JSON 中包含关键字段
	jsonStr := string(data)
	if !strings.Contains(jsonStr, "read_file") {
		t.Error("JSON 中应包含 'read_file'")
	}
	if !strings.Contains(jsonStr, "path") {
		t.Error("JSON 中应包含 'path' 参数")
	}
}
