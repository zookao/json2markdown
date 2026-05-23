package converter

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
)

type FieldInfo struct {
	Name     string
	Type     string
	Required string
	Desc     string
}

// CommentInfo 存储从注释中提取的信息
type CommentInfo struct {
	Required bool
	Desc     string
}

// OrderedMap 保持插入顺序的map
type OrderedMap struct {
	Keys   []string
	Values map[string]interface{}
}

func NewOrderedMap() *OrderedMap {
	return &OrderedMap{
		Values: make(map[string]interface{}),
	}
}

// ConvertJSONToMarkdown 将JSON转换为Markdown表格
func ConvertJSONToMarkdown(jsonStr string) (string, error) {
	// 先提取注释信息
	comments := extractComments(jsonStr)

	// 移除注释后再解析JSON
	cleanJSON := removeComments(jsonStr)

	// 使用OrderedMap解析保持顺序
	orderedData := NewOrderedMap()
	if err := parseObject(json.RawMessage(cleanJSON), orderedData); err != nil {
		return "", fmt.Errorf("JSON解析失败: %v", err)
	}

	fields := parseOrderedJSON(orderedData, "", comments)

	// 生成Markdown表格
	var sb strings.Builder
	sb.WriteString("| 参数名 | 类型 | 是否必填 | 参数说明 |\n")
	sb.WriteString("|--------|------|----------|----------|\n")

	for _, f := range fields {
		sb.WriteString(fmt.Sprintf("| %s | %s | %s | %s |\n", f.Name, f.Type, f.Required, f.Desc))
	}

	return sb.String(), nil
}

// parseObject 解析JSON对象保持键顺序
func parseObject(raw json.RawMessage, om *OrderedMap) error {
	decoder := json.NewDecoder(strings.NewReader(string(raw)))

	token, err := decoder.Token()
	if err != nil {
		return err
	}
	if token != json.Delim('{') {
		return fmt.Errorf("expected { but got %v", token)
	}

	for decoder.More() {
		keyToken, err := decoder.Token()
		if err != nil {
			return err
		}
		key, ok := keyToken.(string)
		if !ok {
			return fmt.Errorf("expected string key but got %v", keyToken)
		}

		var value json.RawMessage
		if err := decoder.Decode(&value); err != nil {
			return err
		}

		// 判断value类型
		trimmed := strings.TrimSpace(string(value))
		if len(trimmed) > 0 && trimmed[0] == '{' {
			// 是对象，递归解析
			nestedOm := NewOrderedMap()
			if err := parseObject(value, nestedOm); err != nil {
				return err
			}
			om.Keys = append(om.Keys, key)
			om.Values[key] = nestedOm
		} else if len(trimmed) > 0 && trimmed[0] == '[' {
			// 是数组，解析数组元素
			arr, err := parseArray(value)
			if err != nil {
				return err
			}
			om.Keys = append(om.Keys, key)
			om.Values[key] = arr
		} else {
			// 基本类型
			var basic interface{}
			if err := json.Unmarshal(value, &basic); err != nil {
				return err
			}
			om.Keys = append(om.Keys, key)
			om.Values[key] = basic
		}
	}

	return nil
}

// parseArray 解析JSON数组
func parseArray(raw json.RawMessage) ([]interface{}, error) {
	var arr []interface{}
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, err
	}

	// 转换数组中的对象为OrderedMap
	for i, item := range arr {
		if obj, ok := item.(map[string]interface{}); ok {
			// 将map转换为OrderedMap
			om := NewOrderedMap()
			// 重新解析保持顺序
			itemRaw, _ := json.Marshal(obj)
			if err := parseObject(json.RawMessage(itemRaw), om); err != nil {
				// 如果失败，使用原始map
				arr[i] = obj
			} else {
				arr[i] = om
			}
		}
	}

	return arr, nil
}

// extractComments 从JSON字符串中提取注释信息
func extractComments(jsonStr string) map[string]CommentInfo {
	comments := make(map[string]CommentInfo)

	// 匹配 "key": value, //注释 格式（基本类型）
	re := regexp.MustCompile(`"([^"]+)"\s*:\s*[^,\n{}[\]]+,\s*//\s*(.+)`)
	matches := re.FindAllStringSubmatch(jsonStr, -1)
	for _, match := range matches {
		if len(match) >= 3 {
			key := match[1]
			comment := strings.TrimSpace(match[2])
			info := parseComment(comment)
			comments[key] = info
		}
	}

	// 匹配对象后的注释: "key": { ... }, //注释 或 "key": { ... } //注释
	// 使用 (?s) 让 . 匹配换行符
	reObj := regexp.MustCompile(`(?s)"([^"]+)"\s*:\s*\{.*?\}\s*,?\s*//\s*(.+)`)
	matchesObj := reObj.FindAllStringSubmatch(jsonStr, -1)
	for _, match := range matchesObj {
		if len(match) >= 3 {
			key := match[1]
			comment := strings.TrimSpace(match[2])
			info := parseComment(comment)
			comments[key] = info
		}
	}

	// 匹配数组后的注释: "key": [ ... ], //注释 或 "key": [ ... ] //注释
	reArr := regexp.MustCompile(`(?s)"([^"]+)"\s*:\s*\[.*?\]\s*,?\s*//\s*(.+)`)
	matchesArr := reArr.FindAllStringSubmatch(jsonStr, -1)
	for _, match := range matchesArr {
		if len(match) >= 3 {
			key := match[1]
			comment := strings.TrimSpace(match[2])
			info := parseComment(comment)
			comments[key] = info
		}
	}

	return comments
}

// parseComment 解析注释内容，提取必填信息和描述
func parseComment(comment string) CommentInfo {
	info := CommentInfo{
		Required: true,
		Desc:     comment,
	}

	// 检查是否包含"非必填"或"选填"
	if strings.Contains(comment, "非必填") || strings.Contains(comment, "选填") || strings.Contains(comment, "可选") {
		info.Required = false
	}

	// 提取描述部分（去掉必填/非必填标记和逗号）
	desc := comment
	// 支持中文逗号和英文逗号
	for _, marker := range []string{"必填，", "必填,", "非必填，", "非必填,", "选填，", "选填,", "可选，", "可选,"} {
		if strings.HasPrefix(desc, marker) {
			desc = strings.TrimPrefix(desc, marker)
			break
		}
	}
	desc = strings.TrimSpace(desc)
	if desc != "" {
		info.Desc = desc
	}

	return info
}

// removeComments 移除JSON中的注释
func removeComments(jsonStr string) string {
	// 移除行尾的 // 注释
	re := regexp.MustCompile(`,\s*//[^\n]*`)
	result := re.ReplaceAllString(jsonStr, ",")

	// 处理最后一个字段后有注释的情况（没有逗号）
	re2 := regexp.MustCompile(`([}\]])\s*//[^\n]*`)
	result = re2.ReplaceAllString(result, "$1")

	return result
}

// parseOrderedJSON 递归解析OrderedMap，提取字段信息
func parseOrderedJSON(om *OrderedMap, prefix string, comments map[string]CommentInfo) []FieldInfo {
	var fields []FieldInfo

	for _, key := range om.Keys {
		value := om.Values[key]

		var fieldName string
		if prefix == "" {
			fieldName = key
		} else {
			fieldName = fmt.Sprintf(`%s["%s"]`, prefix, key)
		}

		commentInfo := getCommentInfo(key, comments)
		subFields := parseValue(value, fieldName, commentInfo.Required, commentInfo.Desc, comments)
		fields = append(fields, subFields...)
	}

	return fields
}

// parseValue 解析JSON值
func parseValue(value interface{}, prefix string, required bool, comment string, comments map[string]CommentInfo) []FieldInfo {
	var fields []FieldInfo

	switch v := value.(type) {
	case *OrderedMap:
		// 对象类型 - 使用OrderedMap保持顺序
		for _, key := range v.Keys {
			val := v.Values[key]
			fieldName := fmt.Sprintf(`%s["%s"]`, prefix, key)
			commentInfo := getCommentInfo(key, comments)
			subFields := parseValue(val, fieldName, commentInfo.Required, commentInfo.Desc, comments)
			fields = append(fields, subFields...)
		}

	case []interface{}:
		// 数组类型
		if len(v) > 0 {
			// 取第一个元素作为示例
			firstItem := v[0]
			switch item := firstItem.(type) {
			case *OrderedMap:
				// 数组元素是对象
				for _, key := range item.Keys {
					val := item.Values[key]
					fieldName := fmt.Sprintf(`%s[0]["%s"]`, prefix, key)
					commentInfo := getCommentInfo(key, comments)
					subFields := parseValue(val, fieldName, commentInfo.Required, commentInfo.Desc, comments)
					fields = append(fields, subFields...)
				}
			case map[string]interface{}:
				// 兼容普通map
				for key, val := range item {
					fieldName := fmt.Sprintf(`%s[0]["%s"]`, prefix, key)
					commentInfo := getCommentInfo(key, comments)
					subFields := parseValue(val, fieldName, commentInfo.Required, commentInfo.Desc, comments)
					fields = append(fields, subFields...)
				}
			default:
				// 数组元素是基本类型
				fieldType := getType(firstItem)
				fields = append(fields, FieldInfo{
					Name:     fmt.Sprintf("%s[0]", prefix),
					Type:     fieldType,
					Required: boolToRequired(required),
					Desc:     comment,
				})
			}
		} else {
			// 空数组
			fields = append(fields, FieldInfo{
				Name:     prefix,
				Type:     "array",
				Required: boolToRequired(required),
				Desc:     comment,
			})
		}

	default:
		// 基本类型
		fieldType := getType(value)
		fields = append(fields, FieldInfo{
			Name:     prefix,
			Type:     fieldType,
			Required: boolToRequired(required),
			Desc:     comment,
		})
	}

	return fields
}

// getCommentInfo 获取字段的注释信息
func getCommentInfo(key string, comments map[string]CommentInfo) CommentInfo {
	if info, ok := comments[key]; ok {
		return info
	}

	// 默认值
	return CommentInfo{
		Required: true,
		Desc:     key,
	}
}

// getType 获取JSON值的类型字符串
func getType(value interface{}) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case bool:
		return "boolean"
	case float64:
		// JSON数字在Go中默认为float64
		if v == float64(int64(v)) {
			return "int"
		}
		return "float"
	case string:
		return "string"
	case []interface{}:
		return "array"
	case map[string]interface{}:
		return "object"
	case *OrderedMap:
		return "object"
	default:
		return "unknown"
	}
}

// boolToRequired 将布尔值转换为"必填"/"非必填"
func boolToRequired(required bool) string {
	if required {
		return "必填"
	}
	return "非必填"
}
