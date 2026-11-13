package main

import (
	"regexp"
	"strings"
)

var wikiLinkPattern = regexp.MustCompile(`\[\[([^\[\]\r\n]+)\]\]`)

var mdLinkPattern = regexp.MustCompile(`\[([^\]\r\n]+)\]\((https?://[^)\s]+)\)`)

var hashtagPattern = regexp.MustCompile(`(^|[\s(])#([A-Za-z][A-Za-z0-9_\-/]*)`)

var orderedListPattern = regexp.MustCompile(`^\d+\.\s`)

var openLinkPattern = regexp.MustCompile(`\[\[([^\[\]]*)$`)

type inlinePart struct {
	Text     string
	TargetID string
	URL      string
	Tag      string
	IsLink   bool
}

func parseWikiLinks(text string, memos []memo) []inlinePart {
	indices := wikiLinkPattern.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return []inlinePart{{Text: text}}
	}

	parts := make([]inlinePart, 0, len(indices)*2+1)
	position := 0
	for _, index := range indices {
		if index[0] > position {
			parts = append(parts, inlinePart{Text: text[position:index[0]]})
		}
		inner := text[index[2]:index[3]]
		targetRaw, display := inner, inner
		if i := strings.Index(inner, "|"); i >= 0 {
			targetRaw, display = inner[:i], inner[i+1:]
		}
		if strings.TrimSpace(display) == "" {
			display = strings.TrimSpace(targetRaw)
		}
		target := normalizeTitle(targetRaw)
		if target == "" {
			parts = append(parts, inlinePart{Text: text[index[0]:index[1]]})
			position = index[1]
			continue
		}
		part := inlinePart{Text: display, IsLink: true}
		for _, item := range memos {
			if normalizeTitle(item.Title) == target {
				part.TargetID = item.ID
				break
			}
		}
		parts = append(parts, part)
		position = index[1]
	}
	if position < len(text) {
		parts = append(parts, inlinePart{Text: text[position:]})
	}
	return parts
}

func parseInline(text string, memos []memo) []inlinePart {
	indices := mdLinkPattern.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return parseRichSegment(text, memos)
	}
	parts := make([]inlinePart, 0, len(indices)*2+1)
	position := 0
	for _, index := range indices {
		if index[0] > position {
			parts = append(parts, parseRichSegment(text[position:index[0]], memos)...)
		}
		parts = append(parts, inlinePart{Text: text[index[2]:index[3]], URL: text[index[4]:index[5]], IsLink: true})
		position = index[1]
	}
	if position < len(text) {
		parts = append(parts, parseRichSegment(text[position:], memos)...)
	}
	return parts
}

func parseRichSegment(text string, memos []memo) []inlinePart {
	parts := make([]inlinePart, 0, 4)
	for _, part := range parseWikiLinks(text, memos) {
		if part.IsLink {
			parts = append(parts, part)
			continue
		}
		parts = append(parts, parseHashtags(part.Text)...)
	}
	return parts
}

func parseHashtags(text string) []inlinePart {
	indices := hashtagPattern.FindAllStringSubmatchIndex(text, -1)
	if len(indices) == 0 {
		return []inlinePart{{Text: text}}
	}
	parts := make([]inlinePart, 0, len(indices)*2+1)
	position := 0
	for _, index := range indices {
		start := index[3]
		if start > position {
			parts = append(parts, inlinePart{Text: text[position:start]})
		}
		parts = append(parts, inlinePart{Text: "#" + text[index[4]:index[5]], Tag: text[index[4]:index[5]], IsLink: true})
		position = index[1]
	}
	if position < len(text) {
		parts = append(parts, inlinePart{Text: text[position:]})
	}
	return parts
}

func extractInlineTags(text string) []string {
	var tags []string
	seen := make(map[string]bool)
	for _, part := range parseHashtags(text) {
		if part.Tag == "" || seen[part.Tag] {
			continue
		}
		seen[part.Tag] = true
		tags = append(tags, part.Tag)
	}
	return tags
}

func (a *app) linkCompletionQuery() (string, bool) {
	var src string
	if !a.composerOpen {
		return "", false
	}
	if a.composerMode == composerModeLive && a.editingBlock >= 0 {
		src = a.editingDraft
	} else if a.composerMode == composerModeSource {
		src = a.composerContent
	} else {
		return "", false
	}
	match := openLinkPattern.FindStringSubmatch(src)
	if match == nil {
		return "", false
	}
	query := match[1]
	if i := strings.Index(query, "|"); i >= 0 {
		query = query[:i]
	}
	return query, true
}

func (a *app) linkSuggestions(query string) []memo {
	q := strings.ToLower(query)
	suggestions := make([]memo, 0, 6)
	for _, item := range a.memos {
		if q == "" || strings.Contains(strings.ToLower(item.Title), q) {
			suggestions = append(suggestions, item)
			if len(suggestions) >= 6 {
				break
			}
		}
	}
	return suggestions
}

func (a *app) acceptLinkSuggestion(title string) {
	acceptInto := func(draft string) string {
		i := strings.LastIndex(draft, "[[")
		if i < 0 {
			return draft
		}
		rest := draft[i+2:]
		if j := strings.Index(rest, "|"); j >= 0 {
			return draft[:i] + "[[" + title + rest[j:]
		}
		return draft[:i] + "[[" + title + "]]"
	}
	if a.composerMode == composerModeLive && a.editingBlock >= 0 {
		a.editingDraft = acceptInto(a.editingDraft)
		return
	}
	if a.composerMode == composerModeSource {
		a.composerContent = acceptInto(a.composerContent)
	}
}
