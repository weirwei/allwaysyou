package memory

import "regexp"

// Key signal patterns that indicate important information worth extracting
var keySignalPatterns = []*regexp.Regexp{
	// Identity - 身份信息
	regexp.MustCompile(`我(是|叫|名字是|名叫)`),
	regexp.MustCompile(`(?i)my name is`),
	regexp.MustCompile(`(?i)i('m| am) (a |an )?[\w]+`), // "I'm a developer", "I am an engineer"

	// Preferences - 偏好
	regexp.MustCompile(`我(喜欢|偏好|习惯|爱|讨厌|不喜欢)`),
	regexp.MustCompile(`(?i)i (like|prefer|love|hate|enjoy|always use)`),

	// Long-term habits - 长期习惯
	regexp.MustCompile(`(以后都|一直|总是|从不|每次都|通常)`),
	regexp.MustCompile(`(?i)(always|never|usually|every time)`),

	// Explicit memory requests - 显式记忆请求
	regexp.MustCompile(`(记住|记得|别忘了|帮我记|请记住)`),
	regexp.MustCompile(`(?i)(remember|don't forget|keep in mind)`),

	// Personal info - 个人信息
	regexp.MustCompile(`我(住在|来自|在.{1,10}工作|是.{1,10}人)`),
	regexp.MustCompile(`我的(职业|工作|专业|年龄|生日)`),
	regexp.MustCompile(`(?i)i (live in|work at|come from|am from)`),

	// Background/context - 背景信息
	regexp.MustCompile(`我(用的是|使用|正在学|在做)`),
	regexp.MustCompile(`我的(项目|代码|系统|应用)`),
}

// ShouldTriggerExtraction checks if the user message contains key signals
// that indicate important information worth extracting for memory
func ShouldTriggerExtraction(userMsg string) bool {
	for _, pattern := range keySignalPatterns {
		if pattern.MatchString(userMsg) {
			return true
		}
	}
	return false
}

// GetMatchedSignals returns all matched signal patterns (for debugging)
func GetMatchedSignals(userMsg string) []string {
	var matched []string
	for _, pattern := range keySignalPatterns {
		if pattern.MatchString(userMsg) {
			matched = append(matched, pattern.String())
		}
	}
	return matched
}
