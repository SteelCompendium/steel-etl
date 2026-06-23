package site

import (
	"fmt"
	"html"
	"strings"
)

// cardHeadSlot is one slot of the shared 6-slot card header. HTML is the
// already-safe inner HTML (callers escape, or pass rich inline HTML); an empty
// HTML omits the slot. Style selects the render: "line" (default), "chip", or
// "mini" (the mini-title).
type cardHeadSlot struct {
	HTML  string
	Style string
}

func hLine(h string) cardHeadSlot { return cardHeadSlot{HTML: h, Style: "line"} }
func hChip(h string) cardHeadSlot { return cardHeadSlot{HTML: h, Style: "chip"} }
func hMini(h string) cardHeadSlot { return cardHeadSlot{HTML: h, Style: "mini"} }

// cardHeadSlots is the full 6-slot header model (left/right × eyebrow/primary/
// deck). LeftPrimary is the name (present on every card). NameTag is the
// heading element for the name ("h3" default). Crest is optional crest HTML
// rendered beside the left column. RoleKey, when set, is emitted as data-role
// on right-primary for accent coloring.
type cardHeadSlots struct {
	Crest, RoleKey, NameTag               string
	LeftEyebrow, LeftPrimary, LeftDeck    cardHeadSlot
	RightEyebrow, RightPrimary, RightDeck cardHeadSlot
}

// renderCardHead emits the contiguous (no blank-line) <header class="sc-head">
// so md_in_html passes it through verbatim.
func renderCardHead(s cardHeadSlots) string {
	nameTag := s.NameTag
	if nameTag == "" {
		nameTag = "h3"
	}
	var b strings.Builder
	b.WriteString(`<header class="sc-head">`)

	b.WriteString(`<div class="sc-head__stack">`)
	if s.Crest != "" {
		b.WriteString(s.Crest)
	}
	b.WriteString(`<div class="sc-head__col sc-head__col--left">`)
	writeCardHeadSlot(&b, "left-eyebrow", "div", s.LeftEyebrow, "")
	writeCardHeadSlot(&b, "left-primary", nameTag, s.LeftPrimary, "")
	writeCardHeadSlot(&b, "left-deck", "div", s.LeftDeck, "")
	b.WriteString(`</div></div>`)

	b.WriteString(`<div class="sc-head__rail sc-head__col--right">`)
	writeCardHeadSlot(&b, "right-eyebrow", "div", s.RightEyebrow, "")
	writeCardHeadSlot(&b, "right-primary", "div", s.RightPrimary, s.RoleKey)
	writeCardHeadSlot(&b, "right-deck", "div", s.RightDeck, "")
	b.WriteString(`</div>`)

	b.WriteString(`</header>`)
	return b.String()
}

// writeCardHeadSlot writes one slot element if it has content. lane is e.g.
// "left-eyebrow"; tag is the element name; roleKey, when non-empty, is emitted
// as data-role.
func writeCardHeadSlot(b *strings.Builder, lane, tag string, sl cardHeadSlot, roleKey string) {
	if strings.TrimSpace(sl.HTML) == "" {
		return
	}
	style := sl.Style
	if style == "" {
		style = "line"
	}
	fmt.Fprintf(b, `<%s class="sc-head__slot sc-head__%s sc-head__slot--%s"`, tag, lane, style)
	if roleKey != "" {
		fmt.Fprintf(b, ` data-role="%s"`, html.EscapeString(roleKey))
	}
	fmt.Fprintf(b, `>%s</%s>`, sl.HTML, tag)
}
