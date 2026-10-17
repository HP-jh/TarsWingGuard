package knowledge

import (
	"strings"
	"unicode"
)

// VoiceCorrector fixes common voice-to-text errors and phonetic substitutions.
type VoiceCorrector struct {
	replacements map[string]string
}

func NewVoiceCorrector() *VoiceCorrector {
	return &VoiceCorrector{
		replacements: defaultReplacements(),
	}
}

func (vc *VoiceCorrector) Correct(input string) string {
	words := strings.Fields(input)
	var result []string
	i := 0
	for i < len(words) {
		matched := false
		// Try longest phrase first (up to 3 words)
		maxN := 3
		if len(words)-i < maxN {
			maxN = len(words) - i
		}
		for n := maxN; n > 0; n-- {
			phrase := strings.ToLower(strings.Join(words[i:i+n], " "))
			if repl, ok := vc.replacements[phrase]; ok {
				result = append(result, preserveCase(words[i], repl))
				i += n
				matched = true
				break
			}
			// Also try concatenated form for single words
			if n == 1 {
				concat := strings.ToLower(words[i])
				if repl, ok := vc.replacements[concat]; ok {
					result = append(result, preserveCase(words[i], repl))
					i++
					matched = true
					break
				}
			}
		}
		if !matched {
			result = append(result, words[i])
			i++
		}
	}
	return strings.Join(result, " ")
}

func preserveCase(original, replacement string) string {
	if len(original) == 0 || len(replacement) == 0 {
		return replacement
	}
	// If original was all upper, return all upper
	if strings.ToUpper(original) == original {
		return strings.ToUpper(replacement)
	}
	// If original started with upper, title-case replacement
	if unicode.IsUpper(rune(original[0])) {
		return strings.Title(replacement)
	}
	return replacement
}

func defaultReplacements() map[string]string {
	return map[string]string{
		// Common voice-to-text errors in Chinese pinyin / phonetic
		"nihao":      "你好",
		"ni hao":     "你好",
		"xiexie":     "谢谢",
		"xie xie":    "谢谢",
		"zaian":      "早安",
		"zai an":     "早安",
		"wanan":      "晚安",
		"wan an":     "晚安",
		"duibuqi":    "对不起",
		"dui bu qi":  "对不起",
		"qingwen":    "请问",
		"qing wen":   "请问",
		"haode":      "好的",
		"hao de":     "好的",
		"meiwenti":   "没问题",
		"mei wenti":  "没问题",
		"mingbai":    "明白",
		"ming bai":   "明白",
		"zhidao":     "知道",
		"zhi dao":    "知道",
		"buzhidao":   "不知道",
		"bu zhidao":  "不知道",
		"keyi":       "可以",
		"ke yi":      "可以",
		"bukeyi":     "不可以",
		"bu keyi":    "不可以",
		"xiexieni":   "谢谢你",
		"xie xie ni": "谢谢你",
		"zaijian":    "再见",
		"zai jian":   "再见",
		// English homophones / common ASR errors
		"teh":        "the",
		"adn":        "and",
		"taht":       "that",
		"wiht":       "with",
		"fo":         "for",
		"ot":         "to",
		"si":         "is",
		"ti":         "it",
		"jsut":       "just",
		"wnat":       "want",
		"hwat":       "what",
		"hwen":       "when",
		"hwere":      "where",
		"hwy":        "why",
		"hwich":      "which",
		"ahve":       "have",
		"hvae":       "have",
		"ta":         "at",
		"oen":        "one",
		"tow":        "two",
		"there":      "their",
		"theyre":     "they're",
		"your":       "you're",
		"its":        "it's",
		"dont":       "don't",
		"wont":       "won't",
		"cant":       "can't",
		"isnt":       "isn't",
		"wasnt":      "wasn't",
		"werent":     "weren't",
		"hasnt":      "hasn't",
		"havent":     "haven't",
		"hadnt":      "hadn't",
		"wouldnt":    "wouldn't",
		"couldnt":    "couldn't",
		"shouldnt":   "shouldn't",
		"didnt":      "didn't",
		"doesnt":     "doesn't",
	}
}
