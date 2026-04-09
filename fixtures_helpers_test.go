package taffy

import (
	"taffy-go/lib"
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
	"taffy-go/geometry"
	"taffy-go/style"
	treepkg "taffy-go/tree"
)

// ---------------------------------------------------------------------------
// Test data structures
// ---------------------------------------------------------------------------

// testFixture represents a parsed XML test fixture.
type testFixture struct {
	Name        string
	UseRounding bool
	Viewport    geometry.Size[style.AvailableSpace]
	InputRoot   *testNode
	Expected    *expectedNode
}

// testNode represents a parsed input <div> or <text> element.
type testNode struct {
	Attrs    map[string]string
	Children []*testNode
	Text     string // for leaf nodes with text content
}

// expectedNode represents a parsed <node> expectation element.
type expectedNode struct {
	X, Y, Width, Height float32
	Children            []*expectedNode
}

// ---------------------------------------------------------------------------
// XML Parsing
// ---------------------------------------------------------------------------

// loadFixture parses an XML fixture file from the given path.
func loadFixture(path string) *testFixture {
	data, err := os.ReadFile(path)
	if err != nil {
		panic(fmt.Sprintf("loadFixture: failed to read %s: %v", path, err))
	}

	decoder := xml.NewDecoder(bytes.NewReader(data))
	fixture := &testFixture{
		UseRounding: true,
		Viewport: geometry.Size[style.AvailableSpace]{
			Width:  style.AvailableSpaceMaxContentValue,
			Height: style.AvailableSpaceMaxContentValue,
		},
	}

	// Find the <test> element
	for {
		tok, err := decoder.Token()
		if err != nil {
			panic(fmt.Sprintf("loadFixture: error parsing XML: %v", err))
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "test" {
			for _, attr := range se.Attr {
				switch attr.Name.Local {
				case "name":
					fixture.Name = attr.Value
				case "use-rounding":
					fixture.UseRounding = attr.Value != "false"
				}
			}
			break
		}
	}

	// Parse children of <test>: <viewport>, <input>, <expectations>
	for {
		tok, err := decoder.Token()
		if err != nil {
			break
		}
		se, ok := tok.(xml.StartElement)
		if !ok {
			continue
		}
		switch se.Name.Local {
		case "viewport":
			attrs := attrsToMap(se.Attr)
			fixture.Viewport = parseViewport(attrs)
			decoder.Skip()
		case "input":
			fixture.InputRoot = parseInputChild(decoder)
		case "expectations":
			fixture.Expected = parseExpectedChild(decoder)
		}
	}

	return fixture
}

// attrsToMap converts XML attributes to a string map.
func attrsToMap(attrs []xml.Attr) map[string]string {
	m := make(map[string]string, len(attrs))
	for _, a := range attrs {
		m[a.Name.Local] = a.Value
	}
	return m
}

// parseInputChild parses the first child element inside <input> recursively.
func parseInputChild(decoder *xml.Decoder) *testNode {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil
		}
		if se, ok := tok.(xml.StartElement); ok {
			return parseInputElement(decoder, se)
		}
	}
}

// parseInputElement parses a <div> or <text> element and its children recursively.
func parseInputElement(decoder *xml.Decoder, start xml.StartElement) *testNode {
	node := &testNode{
		Attrs: attrsToMap(start.Attr),
	}

	var textBuf strings.Builder
	for {
		tok, err := decoder.Token()
		if err != nil {
			return node
		}
		switch t := tok.(type) {
		case xml.StartElement:
			child := parseInputElement(decoder, t)
			node.Children = append(node.Children, child)
		case xml.CharData:
			textBuf.Write(t)
		case xml.EndElement:
			if text := strings.TrimSpace(textBuf.String()); text != "" && len(node.Children) == 0 {
				node.Text = text
			}
			return node
		}
	}
}

// parseExpectedChild parses the first <node> element inside <expectations>.
func parseExpectedChild(decoder *xml.Decoder) *expectedNode {
	for {
		tok, err := decoder.Token()
		if err != nil {
			return nil
		}
		if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "node" {
			return parseExpectedElement(decoder, se)
		}
	}
}

// parseExpectedElement parses a <node> element and its children recursively.
func parseExpectedElement(decoder *xml.Decoder, start xml.StartElement) *expectedNode {
	attrs := attrsToMap(start.Attr)
	node := &expectedNode{
		X:      parseFloat32(attrs["x"]),
		Y:      parseFloat32(attrs["y"]),
		Width:  parseFloat32(attrs["width"]),
		Height: parseFloat32(attrs["height"]),
	}

	for {
		tok, err := decoder.Token()
		if err != nil {
			return node
		}
		switch t := tok.(type) {
		case xml.StartElement:
			if t.Name.Local == "node" {
				child := parseExpectedElement(decoder, t)
				node.Children = append(node.Children, child)
			} else {
				decoder.Skip()
			}
		case xml.EndElement:
			return node
		}
	}
}

// parseFloat32 parses a string as float32.
func parseFloat32(s string) float32 {
	if s == "" {
		return 0
	}
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(fmt.Sprintf("parseFloat32: cannot parse %q: %v", s, err))
	}
	return float32(v)
}

// ---------------------------------------------------------------------------
// Viewport Parsing
// ---------------------------------------------------------------------------

// parseViewport builds geometry.Size[style.AvailableSpace] from viewport attributes.
func parseViewport(attrs map[string]string) geometry.Size[style.AvailableSpace] {
	return geometry.Size[style.AvailableSpace]{
		Width:  parseAvailableSpace(attrs["width"]),
		Height: parseAvailableSpace(attrs["height"]),
	}
}

// parseAvailableSpace parses a viewport dimension string.
func parseAvailableSpace(s string) style.AvailableSpace {
	switch s {
	case "", "max-content":
		return style.AvailableSpaceMaxContentValue
	case "min-content":
		return style.AvailableSpaceMinContentValue
	default:
		val := parsePxValue(s)
		return style.AvailableSpaceFromDefinite(val)
	}
}

// parsePxValue parses a value like "500px" or "500" as float32.
func parsePxValue(s string) float32 {
	s = strings.TrimSuffix(s, "px")
	v, err := strconv.ParseFloat(s, 32)
	if err != nil {
		panic(fmt.Sprintf("parsePxValue: cannot parse %q: %v", s, err))
	}
	return float32(v)
}

// ---------------------------------------------------------------------------
// Attribute-to-style.Style Mapping
// ---------------------------------------------------------------------------

// buildStyle creates a style.Style from a map of XML attributes.
func buildStyle(attrs map[string]string) style.Style {
	s := style.DefaultStyle()

	// style.Display
	if v, ok := attrs["display"]; ok {
		s.Display = parseDisplay(v)
	}

	// style.Direction
	if v, ok := attrs["direction"]; ok {
		s.Direction = parseDirection(v)
	}

	// style.BoxSizing
	if v, ok := attrs["box-sizing"]; ok {
		s.BoxSizing = parseBoxSizing(v)
	}

	// style.Position
	if v, ok := attrs["position"]; ok {
		s.Position = parsePosition(v)
	}

	// style.Overflow
	if v, ok := attrs["overflow-x"]; ok {
		s.Overflow.X = parseOverflow(v)
	}
	if v, ok := attrs["overflow-y"]; ok {
		s.Overflow.Y = parseOverflow(v)
	}

	// Scrollbar width
	if v, ok := attrs["scrollbar-width"]; ok {
		s.ScrollbarWidth = parsePxValue(v)
	}

	// style.Float
	if v, ok := attrs["float"]; ok {
		s.Float = parseFloat(v)
	}

	// style.Clear
	if v, ok := attrs["clear"]; ok {
		s.Clear = parseClear(v)
	}

	// Size
	if v, ok := attrs["width"]; ok {
		s.Size.Width = parseDimension(v)
	}
	if v, ok := attrs["height"]; ok {
		s.Size.Height = parseDimension(v)
	}

	// Min size
	if v, ok := attrs["min-width"]; ok {
		s.MinSize.Width = parseDimension(v)
	}
	if v, ok := attrs["min-height"]; ok {
		s.MinSize.Height = parseDimension(v)
	}

	// Max size
	if v, ok := attrs["max-width"]; ok {
		s.MaxSize.Width = parseDimension(v)
	}
	if v, ok := attrs["max-height"]; ok {
		s.MaxSize.Height = parseDimension(v)
	}

	// Inset
	if v, ok := attrs["top"]; ok {
		s.Inset.Top = parseLPA(v)
	}
	if v, ok := attrs["left"]; ok {
		s.Inset.Left = parseLPA(v)
	}
	if v, ok := attrs["bottom"]; ok {
		s.Inset.Bottom = parseLPA(v)
	}
	if v, ok := attrs["right"]; ok {
		s.Inset.Right = parseLPA(v)
	}

	// Margin
	if v, ok := attrs["margin-top"]; ok {
		s.Margin.Top = parseLPA(v)
	}
	if v, ok := attrs["margin-left"]; ok {
		s.Margin.Left = parseLPA(v)
	}
	if v, ok := attrs["margin-bottom"]; ok {
		s.Margin.Bottom = parseLPA(v)
	}
	if v, ok := attrs["margin-right"]; ok {
		s.Margin.Right = parseLPA(v)
	}

	// Padding
	if v, ok := attrs["padding-top"]; ok {
		s.Padding.Top = parseLP(v)
	}
	if v, ok := attrs["padding-left"]; ok {
		s.Padding.Left = parseLP(v)
	}
	if v, ok := attrs["padding-bottom"]; ok {
		s.Padding.Bottom = parseLP(v)
	}
	if v, ok := attrs["padding-right"]; ok {
		s.Padding.Right = parseLP(v)
	}

	// Border
	if v, ok := attrs["border-top"]; ok {
		s.Border.Top = parseLP(v)
	}
	if v, ok := attrs["border-left"]; ok {
		s.Border.Left = parseLP(v)
	}
	if v, ok := attrs["border-bottom"]; ok {
		s.Border.Bottom = parseLP(v)
	}
	if v, ok := attrs["border-right"]; ok {
		s.Border.Right = parseLP(v)
	}

	// Gap
	if v, ok := attrs["column-gap"]; ok {
		s.Gap.Width = parseLP(v)
	}
	if v, ok := attrs["row-gap"]; ok {
		s.Gap.Height = parseLP(v)
	}

	// Aspect ratio
	if v, ok := attrs["aspect-ratio"]; ok {
		val, err := strconv.ParseFloat(v, 32)
		if err == nil {
			s.AspectRatio = lib.Some(float32(val))
		}
	}

	// Alignment (optional)
	if v, ok := attrs["align-items"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.AlignItems = lib.Some(ai)
		}
	}
	if v, ok := attrs["align-self"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.AlignSelf = lib.Some(ai)
		}
	}
	if v, ok := attrs["justify-items"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.JustifyItems = lib.Some(ai)
		}
	}
	if v, ok := attrs["justify-self"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.JustifySelf = lib.Some(ai)
		}
	}
	if v, ok := attrs["align-content"]; ok {
		if ac, parsed := parseAlignContent(v); parsed {
			s.AlignContent = lib.Some(ac)
		}
	}
	if v, ok := attrs["justify-content"]; ok {
		if ac, parsed := parseAlignContent(v); parsed {
			s.JustifyContent = lib.Some(ac)
		}
	}

	// Text align
	if v, ok := attrs["text-align"]; ok {
		s.TextAlign = parseTextAlign(v)
	}

	// Flex
	if v, ok := attrs["flex-direction"]; ok {
		s.FlexDirection = parseFlexDirection(v)
	}
	if v, ok := attrs["flex-wrap"]; ok {
		s.FlexWrap = parseFlexWrap(v)
	}
	if v, ok := attrs["flex-grow"]; ok {
		val, _ := strconv.ParseFloat(v, 32)
		s.FlexGrow = float32(val)
	}
	if v, ok := attrs["flex-shrink"]; ok {
		val, _ := strconv.ParseFloat(v, 32)
		s.FlexShrink = float32(val)
	}
	if v, ok := attrs["flex-basis"]; ok {
		s.FlexBasis = parseDimension(v)
	}

	// Grid auto flow
	if v, ok := attrs["grid-auto-flow"]; ok {
		s.GridAutoFlow = parseGridAutoFlow(v)
	}

	// Grid template
	if v, ok := attrs["grid-template-rows"]; ok {
		s.GridTemplateRows = parseGridTemplate(v)
	}
	if v, ok := attrs["grid-template-columns"]; ok {
		s.GridTemplateColumns = parseGridTemplate(v)
	}

	// Grid auto tracks
	if v, ok := attrs["grid-auto-rows"]; ok {
		s.GridAutoRows = parseGridAutoTracks(v)
	}
	if v, ok := attrs["grid-auto-columns"]; ok {
		s.GridAutoColumns = parseGridAutoTracks(v)
	}

	// Grid placement
	if v, ok := attrs["grid-row-start"]; ok {
		s.GridRow.Start = parseGridPlacement(v)
	}
	if v, ok := attrs["grid-row-end"]; ok {
		s.GridRow.End = parseGridPlacement(v)
	}
	if v, ok := attrs["grid-column-start"]; ok {
		s.GridColumn.Start = parseGridPlacement(v)
	}
	if v, ok := attrs["grid-column-end"]; ok {
		s.GridColumn.End = parseGridPlacement(v)
	}

	return s
}

// ---------------------------------------------------------------------------
// Individual value parsers
// ---------------------------------------------------------------------------

func parseDisplay(s string) style.Display {
	switch s {
	case "flex":
		return style.DisplayFlex
	case "grid":
		return style.DisplayGrid
	case "block":
		return style.DisplayBlock
	case "none":
		return style.DisplayNone
	default:
		return style.DisplayFlex
	}
}

func parseDirection(s string) style.Direction {
	switch s {
	case "rtl":
		return style.DirectionRTL
	default:
		return style.DirectionLTR
	}
}

func parseBoxSizing(s string) style.BoxSizing {
	switch s {
	case "content-box":
		return style.BoxSizingContentBox
	default:
		return style.BoxSizingBorderBox
	}
}

func parsePosition(s string) style.Position {
	switch s {
	case "absolute":
		return style.PositionAbsolute
	default:
		return style.PositionRelative
	}
}

func parseOverflow(s string) style.Overflow {
	switch s {
	case "hidden":
		return style.OverflowHidden
	case "scroll":
		return style.OverflowScroll
	case "clip":
		return style.OverflowClip
	default:
		return style.OverflowVisible
	}
}

func parseFloat(s string) style.Float {
	switch s {
	case "left":
		return style.FloatLeft
	case "right":
		return style.FloatRight
	default:
		return style.FloatNone
	}
}

func parseClear(s string) style.Clear {
	switch s {
	case "left":
		return style.ClearLeft
	case "right":
		return style.ClearRight
	case "both":
		return style.ClearBoth
	default:
		return style.ClearNone
	}
}

func parseTextAlign(s string) style.TextAlign {
	switch s {
	case "left", "-webkit-left":
		return style.TextAlignLegacyLeft
	case "right", "-webkit-right":
		return style.TextAlignLegacyRight
	case "center", "-webkit-center":
		return style.TextAlignLegacyCenter
	default:
		return style.TextAlignAuto
	}
}

func parseFlexDirection(s string) style.FlexDirection {
	switch s {
	case "column":
		return style.FlexDirectionColumn
	case "row-reverse":
		return style.FlexDirectionRowReverse
	case "column-reverse":
		return style.FlexDirectionColumnReverse
	default:
		return style.FlexDirectionRow
	}
}

func parseFlexWrap(s string) style.FlexWrap {
	switch s {
	case "wrap":
		return style.FlexWrapWrap
	case "wrap-reverse":
		return style.FlexWrapWrapReverse
	default:
		return style.FlexWrapNoWrap
	}
}

func parseAlignItems(s string) (style.AlignItems, bool) {
	switch s {
	case "start":
		return style.AlignItemsStart, true
	case "end":
		return style.AlignItemsEnd, true
	case "flex-start":
		return style.AlignItemsFlexStart, true
	case "flex-end":
		return style.AlignItemsFlexEnd, true
	case "center":
		return style.AlignItemsCenter, true
	case "baseline":
		return style.AlignItemsBaseline, true
	case "stretch":
		return style.AlignItemsStretch, true
	default:
		return 0, false
	}
}

func parseAlignContent(s string) (style.AlignContent, bool) {
	switch s {
	case "start":
		return style.AlignContentStart, true
	case "end":
		return style.AlignContentEnd, true
	case "flex-start":
		return style.AlignContentFlexStart, true
	case "flex-end":
		return style.AlignContentFlexEnd, true
	case "center":
		return style.AlignContentCenter, true
	case "stretch":
		return style.AlignContentStretch, true
	case "space-between":
		return style.AlignContentSpaceBetween, true
	case "space-evenly":
		return style.AlignContentSpaceEvenly, true
	case "space-around":
		return style.AlignContentSpaceAround, true
	default:
		return 0, false
	}
}

func parseGridAutoFlow(s string) style.GridAutoFlow {
	switch s {
	case "column":
		return style.GridAutoFlowColumn
	case "row dense":
		return style.GridAutoFlowRowDense
	case "column dense":
		return style.GridAutoFlowColumnDense
	default:
		return style.GridAutoFlowRow
	}
}

// ---------------------------------------------------------------------------
// Dimension / LP / LPA parsers
// ---------------------------------------------------------------------------

// parseDimension parses a CSS dimension value like "100px", "50%", "auto".
func parseDimension(s string) style.Dimension {
	switch s {
	case "auto":
		return style.Dimension{Tag: style.DimensionAuto}
	case "min-content", "max-content":
		return style.Dimension{Tag: style.DimensionAuto}
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.DimensionFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.DimensionFromLength(float32(val))
}

// parseLPA parses a LengthPercentageAuto value.
func parseLPA(s string) style.LengthPercentageAuto {
	switch s {
	case "auto":
		return style.LengthPercentageAuto{Tag: style.LPAAuto}
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.LPAFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.LPAFromLength(float32(val))
}

// parseLP parses a LengthPercentage value.
func parseLP(s string) style.LengthPercentage {
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.LPFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.LPFromLength(float32(val))
}

// ---------------------------------------------------------------------------
// Grid Template Parsing
// ---------------------------------------------------------------------------

// parseGridTemplate parses a grid-template-rows or grid-template-columns value.
// Examples: "40px 40px 40px", "1fr 2fr", "repeat(3, 1fr)", "minmax(100px, 1fr)"
func parseGridTemplate(s string) []style.GridTemplateComponent {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return nil
	}

	var components []style.GridTemplateComponent
	tokens := tokenizeGridTemplate(s)

	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "repeat(") {
			components = append(components, parseRepeat(tok))
		} else if strings.HasPrefix(tok, "minmax(") {
			tsf := parseMinmax(tok)
			components = append(components, style.GridTemplateSingleVal(tsf))
		} else if strings.HasPrefix(tok, "fit-content(") {
			tsf := parseFitContent(tok)
			components = append(components, style.GridTemplateSingleVal(tsf))
		} else {
			tsf := parseSingleTrack(tok)
			components = append(components, style.GridTemplateSingleVal(tsf))
		}
	}

	return components
}

// parseGridAutoTracks parses a grid-auto-rows or grid-auto-columns value.
// These are space-separated TrackSizingFunction values (no repeat allowed).
func parseGridAutoTracks(s string) []style.TrackSizingFunction {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return nil
	}

	var tracks []style.TrackSizingFunction
	tokens := tokenizeGridTemplate(s)
	for _, tok := range tokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "minmax(") {
			tracks = append(tracks, parseMinmax(tok))
		} else if strings.HasPrefix(tok, "fit-content(") {
			tracks = append(tracks, parseFitContent(tok))
		} else {
			tracks = append(tracks, parseSingleTrack(tok))
		}
	}
	return tracks
}

// tokenizeGridTemplate splits a grid template string into top-level tokens,
// respecting parenthesized groups.
func tokenizeGridTemplate(s string) []string {
	var tokens []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ' ':
			if depth == 0 {
				tok := strings.TrimSpace(s[start:i])
				if tok != "" {
					tokens = append(tokens, tok)
				}
				start = i + 1
			}
		}
	}
	if start < len(s) {
		tok := strings.TrimSpace(s[start:])
		if tok != "" {
			tokens = append(tokens, tok)
		}
	}
	return tokens
}

// parseSingleTrack parses a single track sizing function token.
func parseSingleTrack(s string) style.TrackSizingFunction {
	switch s {
	case "auto":
		return style.TrackSizingAuto
	case "min-content":
		return style.TrackSizingMinContent
	case "max-content":
		return style.TrackSizingMaxContent
	}
	if strings.HasSuffix(s, "fr") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "fr"), 32)
		return style.TrackFromFr(float32(val))
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.TrackFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.TrackFromLength(float32(val))
}

// parseMinmax parses "minmax(min, max)" into a TrackSizingFunction.
func parseMinmax(s string) style.TrackSizingFunction {
	// Strip "minmax(" and trailing ")"
	inner := s[7 : len(s)-1]
	parts := splitComma(inner)
	if len(parts) != 2 {
		panic(fmt.Sprintf("parseMinmax: expected 2 parts, got %d in %q", len(parts), s))
	}
	minPart := strings.TrimSpace(parts[0])
	maxPart := strings.TrimSpace(parts[1])
	return style.TrackSizingFunction{
		Min: parseMinTrackSizing(minPart),
		Max: parseMaxTrackSizing(maxPart),
	}
}

// parseFitContent parses "fit-content(200px)" or "fit-content(50%)" into a TrackSizingFunction.
func parseFitContent(s string) style.TrackSizingFunction {
	inner := s[12 : len(s)-1]
	inner = strings.TrimSpace(inner)
	return style.TrackFitContent(parseLP(inner))
}

// parseRepeat parses "repeat(3, 40px)" or "repeat(auto-fill, minmax(...))" etc.
func parseRepeat(s string) style.GridTemplateComponent {
	// Strip "repeat(" and trailing ")"
	inner := s[7 : len(s)-1]

	// Split on first comma (count, rest)
	commaIdx := strings.Index(inner, ",")
	if commaIdx < 0 {
		panic(fmt.Sprintf("parseRepeat: no comma in %q", s))
	}
	countStr := strings.TrimSpace(inner[:commaIdx])
	tracksStr := strings.TrimSpace(inner[commaIdx+1:])

	count := parseRepetitionCount(countStr)

	// Parse tracks inside the repeat
	var tracks []style.TrackSizingFunction
	trackTokens := tokenizeGridTemplate(tracksStr)
	for _, tok := range trackTokens {
		tok = strings.TrimSpace(tok)
		if tok == "" {
			continue
		}
		if strings.HasPrefix(tok, "minmax(") {
			tracks = append(tracks, parseMinmax(tok))
		} else if strings.HasPrefix(tok, "fit-content(") {
			tracks = append(tracks, parseFitContent(tok))
		} else {
			tracks = append(tracks, parseSingleTrack(tok))
		}
	}

	return style.GridTemplateRepeatVal(style.GridTemplateRepetition{
		Count:  count,
		Tracks: tracks,
	})
}

// parseRepetitionCount parses the count argument in repeat().
func parseRepetitionCount(s string) style.RepetitionCount {
	switch s {
	case "auto-fill":
		return style.RepetitionAutoFillVal
	case "auto-fit":
		return style.RepetitionAutoFitVal
	default:
		n, _ := strconv.Atoi(s)
		return style.RepetitionCountFromU16(uint16(n))
	}
}

// splitComma splits a string on commas, respecting parenthesized groups.
func splitComma(s string) []string {
	var parts []string
	depth := 0
	start := 0
	for i := 0; i < len(s); i++ {
		switch s[i] {
		case '(':
			depth++
		case ')':
			depth--
		case ',':
			if depth == 0 {
				parts = append(parts, s[start:i])
				start = i + 1
			}
		}
	}
	parts = append(parts, s[start:])
	return parts
}

// parseMinTrackSizing parses a min track sizing function string.
func parseMinTrackSizing(s string) style.MinTrackSizingFunction {
	switch s {
	case "auto":
		return style.MinTrackAutoVal
	case "min-content":
		return style.MinTrackMinContentVal
	case "max-content":
		return style.MinTrackMaxContentVal
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.MinTrackFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.MinTrackFromLength(float32(val))
}

// parseMaxTrackSizing parses a max track sizing function string.
func parseMaxTrackSizing(s string) style.MaxTrackSizingFunction {
	switch s {
	case "auto":
		return style.MaxTrackAutoVal
	case "min-content":
		return style.MaxTrackMinContentVal
	case "max-content":
		return style.MaxTrackMaxContentVal
	}
	if strings.HasSuffix(s, "fr") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "fr"), 32)
		return style.MaxTrackFromFr(float32(val))
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return style.MaxTrackFromPercent(float32(val) / 100.0)
	}
	if strings.HasPrefix(s, "fit-content(") {
		inner := s[12 : len(s)-1]
		lp := parseLP(strings.TrimSpace(inner))
		return style.MaxTrackFitContentFromLP(lp)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return style.MaxTrackFromLength(float32(val))
}

// ---------------------------------------------------------------------------
// Grid Placement Parsing
// ---------------------------------------------------------------------------

// parseGridPlacement parses a grid placement value like "1", "span 2", "-1", "auto".
func parseGridPlacement(s string) style.GridPlacement {
	s = strings.TrimSpace(s)
	switch s {
	case "", "auto":
		return style.GridPlacementAutoVal
	}
	if strings.HasPrefix(s, "span ") {
		rest := strings.TrimSpace(s[5:])
		n, _ := strconv.Atoi(rest)
		return style.GridPlacementFromSpan(uint16(n))
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return style.GridPlacementAutoVal
	}
	return style.GridPlacementFromLine(int16(n))
}

// ---------------------------------------------------------------------------
// WritingMode
// ---------------------------------------------------------------------------

type writingMode int

const (
	writingModeHorizontal writingMode = iota
	writingModeVertical
)

func parseWritingMode(s string) writingMode {
	if strings.Contains(s, "vertical") {
		return writingModeVertical
	}
	return writingModeHorizontal
}

// ---------------------------------------------------------------------------
// Tree Building
// ---------------------------------------------------------------------------

// buildTree constructs the TaffyTree node hierarchy from a parsed testNode.
func buildTree(tree *TaffyTree, node *testNode) treepkg.NodeId {
	style := buildStyle(node.Attrs)

	if len(node.Children) > 0 {
		// Internal node: build children first
		childIds := make([]treepkg.NodeId, len(node.Children))
		for i, child := range node.Children {
			childIds[i] = buildTree(tree, child)
		}
		return tree.NewWithChildren(style, childIds)
	}

	if node.Text != "" {
		// Leaf with text content — use Ahem measure function
		wm := writingModeHorizontal
		if v, ok := node.Attrs["writing-mode"]; ok {
			wm = parseWritingMode(v)
		}
		ctx := ahemMeasure(node.Text, wm)
		return tree.NewLeafWithContext(style, ctx)
	}

	// Plain leaf node (no children, no text)
	return tree.NewLeaf(style)
}

// ---------------------------------------------------------------------------
// Ahem Font Measure Function
// ---------------------------------------------------------------------------

const (
	ahemCharWidth  float32 = 10.0
	ahemCharHeight float32 = 10.0
	zwsp                   = '\u200B'
)

// ahemMeasure creates a NodeContext (measure function) that simulates the Ahem test font.
// In Ahem, each character is 10px wide and 10px tall. Lines are separated by zero-width space (U+200B).
func ahemMeasure(text string, wm writingMode) NodeContext {
	return func(
		knownDimensions geometry.Size[lib.Option[float32]],
		parentSize geometry.Size[lib.Option[float32]],
		availableSpace geometry.Size[style.AvailableSpace],
		sizingMode treepkg.SizingMode,
	) geometry.Size[float32] {
		// If both dimensions are known, return them directly
		if knownDimensions.Width.IsSome && knownDimensions.Height.IsSome {
			return geometry.Size[float32]{
				Width:  knownDimensions.Width.Value,
				Height: knownDimensions.Height.Value,
			}
		}

		lines := strings.Split(text, string(zwsp))

		if len(lines) == 0 {
			return geometry.SizeZeroF32
		}

		// Determine inline and block axes based on writing mode
		var inlineAxis, blockAxis geometry.AbsoluteAxis
		if wm == writingModeHorizontal {
			inlineAxis = geometry.AbsoluteAxisHorizontal
			blockAxis = geometry.AbsoluteAxisVertical
		} else {
			inlineAxis = geometry.AbsoluteAxisVertical
			blockAxis = geometry.AbsoluteAxisHorizontal
		}

		// Compute min and max line lengths (in characters)
		minLineLen := 0
		maxLineLen := 0
		for _, line := range lines {
			lineLen := len(line)
			if lineLen > minLineLen {
				minLineLen = lineLen
			}
			maxLineLen += lineLen
		}

		// Known inline dimension
		knownInline := sizeGetAbs(knownDimensions, inlineAxis)
		availInline := sizeAvailableGetAbs(availableSpace, inlineAxis)

		var inlineSize float32
		if knownInline.IsSome {
			inlineSize = knownInline.Value
		} else {
			switch availInline.Tag {
			case style.AvailableSpaceMinContent:
				inlineSize = float32(minLineLen) * ahemCharWidth
			case style.AvailableSpaceMaxContent:
				inlineSize = float32(maxLineLen) * ahemCharWidth
			case style.AvailableSpaceDefinite:
				inlineSize = fmin(availInline.Value, float32(maxLineLen)*ahemCharWidth)
			}
		}
		inlineSize = fmax(inlineSize, float32(minLineLen)*ahemCharWidth)

		// Known block dimension
		knownBlock := sizeGetAbs(knownDimensions, blockAxis)
		var blockSize float32
		if knownBlock.IsSome {
			blockSize = knownBlock.Value
		} else {
			inlineLineLen := int(math.Floor(float64(inlineSize / ahemCharWidth)))
			if inlineLineLen == 0 {
				inlineLineLen = 1
			}
			lineCount := 1
			currentLineLen := 0
			for _, line := range lines {
				ll := len(line)
				if currentLineLen+ll > inlineLineLen {
					if currentLineLen > 0 {
						lineCount++
					}
					currentLineLen = ll
				} else {
					currentLineLen += ll
				}
			}
			blockSize = float32(lineCount) * ahemCharHeight
		}

		if wm == writingModeHorizontal {
			return geometry.Size[float32]{Width: inlineSize, Height: blockSize}
		}
		return geometry.Size[float32]{Width: blockSize, Height: inlineSize}
	}
}

// Helper to get AvailableSpace by absolute axis.
func sizeAvailableGetAbs(s geometry.Size[style.AvailableSpace], axis geometry.AbsoluteAxis) style.AvailableSpace {
	if axis == geometry.AbsoluteAxisHorizontal {
		return s.Width
	}
	return s.Height
}

// Helper to get lib.Option[float32] by absolute axis.
func sizeGetAbs(s geometry.Size[lib.Option[float32]], axis geometry.AbsoluteAxis) lib.Option[float32] {
	if axis == geometry.AbsoluteAxisHorizontal {
		return s.Width
	}
	return s.Height
}

// fmin returns the smaller of two float32 values.
func fmin(a, b float32) float32 {
	if a < b {
		return a
	}
	return b
}

// fmax returns the larger of two float32 values.
func fmax(a, b float32) float32 {
	if a > b {
		return a
	}
	return b
}

// ---------------------------------------------------------------------------
// Test Measure Function (compatible with MeasureFunction type)
// ---------------------------------------------------------------------------

// testMeasureFunction is a MeasureFunction that delegates to the node's NodeContext.
func testMeasureFunction(
	knownDimensions geometry.Size[lib.Option[float32]],
	availableSpace geometry.Size[style.AvailableSpace],
	nodeId treepkg.NodeId,
	context *NodeContext,
	style *style.Style,
) geometry.Size[float32] {
	if knownDimensions.Width.IsSome && knownDimensions.Height.IsSome {
		return geometry.Size[float32]{
			Width:  knownDimensions.Width.Value,
			Height: knownDimensions.Height.Value,
		}
	}

	if context == nil || *context == nil {
		return geometry.Size[float32]{
			Width:  knownDimensions.Width.UnwrapOr(0),
			Height: knownDimensions.Height.UnwrapOr(0),
		}
	}

	computeSize := (*context)(knownDimensions, geometry.SizeOptionNone, availableSpace, treepkg.SizingModeInherentSize)

	return geometry.Size[float32]{
		Width:  knownDimensions.Width.UnwrapOr(computeSize.Width),
		Height: knownDimensions.Height.UnwrapOr(computeSize.Height),
	}
}

// ---------------------------------------------------------------------------
// Layout Assertion
// ---------------------------------------------------------------------------

// assertLayoutMatches recursively checks that the computed layout matches expectations.
func assertLayoutMatches(tree *TaffyTree, nodeId treepkg.NodeId, expected *expectedNode) {
	layout := tree.GetLayout(nodeId)
	const tolerance = 0.1

	Expect(layout.Location.X).To(
		BeNumerically("~", expected.X, tolerance),
		fmt.Sprintf("node %v: X mismatch", nodeId),
	)
	Expect(layout.Location.Y).To(
		BeNumerically("~", expected.Y, tolerance),
		fmt.Sprintf("node %v: Y mismatch", nodeId),
	)
	Expect(layout.Size.Width).To(
		BeNumerically("~", expected.Width, tolerance),
		fmt.Sprintf("node %v: Width mismatch", nodeId),
	)
	Expect(layout.Size.Height).To(
		BeNumerically("~", expected.Height, tolerance),
		fmt.Sprintf("node %v: Height mismatch", nodeId),
	)

	children := tree.Children(nodeId)
	Expect(len(children)).To(
		Equal(len(expected.Children)),
		fmt.Sprintf("node %v: child count mismatch", nodeId),
	)
	for i, child := range children {
		assertLayoutMatches(tree, child, expected.Children[i])
	}
}

// ---------------------------------------------------------------------------
// Running a fixture test
// ---------------------------------------------------------------------------

// runFixtureTest loads a fixture file, builds the tree, computes layout, and asserts.
func runFixtureTest(path string) {
	fixture := loadFixture(path)

	tree := New()
	if fixture.UseRounding {
		tree.EnableRounding()
	} else {
		tree.DisableRounding()
	}

	rootId := buildTree(tree, fixture.InputRoot)
	tree.ComputeLayoutWithMeasure(rootId, fixture.Viewport, testMeasureFunction)

	assertLayoutMatches(tree, rootId, fixture.Expected)
}
