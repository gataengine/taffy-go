package taffy

import (
	"bytes"
	"encoding/xml"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"

	. "github.com/onsi/gomega"
)

// ---------------------------------------------------------------------------
// Test data structures
// ---------------------------------------------------------------------------

// testFixture represents a parsed XML test fixture.
type testFixture struct {
	Name        string
	UseRounding bool
	Viewport    Size[AvailableSpace]
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
		Viewport: Size[AvailableSpace]{
			Width:  AvailableSpaceMaxContentValue,
			Height: AvailableSpaceMaxContentValue,
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

// parseViewport builds Size[AvailableSpace] from viewport attributes.
func parseViewport(attrs map[string]string) Size[AvailableSpace] {
	return Size[AvailableSpace]{
		Width:  parseAvailableSpace(attrs["width"]),
		Height: parseAvailableSpace(attrs["height"]),
	}
}

// parseAvailableSpace parses a viewport dimension string.
func parseAvailableSpace(s string) AvailableSpace {
	switch s {
	case "", "max-content":
		return AvailableSpaceMaxContentValue
	case "min-content":
		return AvailableSpaceMinContentValue
	default:
		val := parsePxValue(s)
		return AvailableSpaceFromDefinite(val)
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
// Attribute-to-Style Mapping
// ---------------------------------------------------------------------------

// buildStyle creates a Style from a map of XML attributes.
func buildStyle(attrs map[string]string) Style {
	s := DefaultStyle()

	// Display
	if v, ok := attrs["display"]; ok {
		s.Display = parseDisplay(v)
	}

	// Direction
	if v, ok := attrs["direction"]; ok {
		s.Direction = parseDirection(v)
	}

	// BoxSizing
	if v, ok := attrs["box-sizing"]; ok {
		s.BoxSizing = parseBoxSizing(v)
	}

	// Position
	if v, ok := attrs["position"]; ok {
		s.Position = parsePosition(v)
	}

	// Overflow
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

	// Float
	if v, ok := attrs["float"]; ok {
		s.Float = parseFloat(v)
	}

	// Clear
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
			s.AspectRatio = Some(float32(val))
		}
	}

	// Alignment (optional)
	if v, ok := attrs["align-items"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.AlignItems = Some(ai)
		}
	}
	if v, ok := attrs["align-self"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.AlignSelf = Some(ai)
		}
	}
	if v, ok := attrs["justify-items"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.JustifyItems = Some(ai)
		}
	}
	if v, ok := attrs["justify-self"]; ok {
		if ai, parsed := parseAlignItems(v); parsed {
			s.JustifySelf = Some(ai)
		}
	}
	if v, ok := attrs["align-content"]; ok {
		if ac, parsed := parseAlignContent(v); parsed {
			s.AlignContent = Some(ac)
		}
	}
	if v, ok := attrs["justify-content"]; ok {
		if ac, parsed := parseAlignContent(v); parsed {
			s.JustifyContent = Some(ac)
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

func parseDisplay(s string) Display {
	switch s {
	case "flex":
		return DisplayFlex
	case "grid":
		return DisplayGrid
	case "block":
		return DisplayBlock
	case "none":
		return DisplayNone
	default:
		return DisplayFlex
	}
}

func parseDirection(s string) Direction {
	switch s {
	case "rtl":
		return DirectionRTL
	default:
		return DirectionLTR
	}
}

func parseBoxSizing(s string) BoxSizing {
	switch s {
	case "content-box":
		return BoxSizingContentBox
	default:
		return BoxSizingBorderBox
	}
}

func parsePosition(s string) Position {
	switch s {
	case "absolute":
		return PositionAbsolute
	default:
		return PositionRelative
	}
}

func parseOverflow(s string) Overflow {
	switch s {
	case "hidden":
		return OverflowHidden
	case "scroll":
		return OverflowScroll
	case "clip":
		return OverflowClip
	default:
		return OverflowVisible
	}
}

func parseFloat(s string) Float {
	switch s {
	case "left":
		return FloatLeft
	case "right":
		return FloatRight
	default:
		return FloatNone
	}
}

func parseClear(s string) Clear {
	switch s {
	case "left":
		return ClearLeft
	case "right":
		return ClearRight
	case "both":
		return ClearBoth
	default:
		return ClearNone
	}
}

func parseTextAlign(s string) TextAlign {
	switch s {
	case "left", "-webkit-left":
		return TextAlignLegacyLeft
	case "right", "-webkit-right":
		return TextAlignLegacyRight
	case "center", "-webkit-center":
		return TextAlignLegacyCenter
	default:
		return TextAlignAuto
	}
}

func parseFlexDirection(s string) FlexDirection {
	switch s {
	case "column":
		return FlexDirectionColumn
	case "row-reverse":
		return FlexDirectionRowReverse
	case "column-reverse":
		return FlexDirectionColumnReverse
	default:
		return FlexDirectionRow
	}
}

func parseFlexWrap(s string) FlexWrap {
	switch s {
	case "wrap":
		return FlexWrapWrap
	case "wrap-reverse":
		return FlexWrapWrapReverse
	default:
		return FlexWrapNoWrap
	}
}

func parseAlignItems(s string) (AlignItems, bool) {
	switch s {
	case "start":
		return AlignItemsStart, true
	case "end":
		return AlignItemsEnd, true
	case "flex-start":
		return AlignItemsFlexStart, true
	case "flex-end":
		return AlignItemsFlexEnd, true
	case "center":
		return AlignItemsCenter, true
	case "baseline":
		return AlignItemsBaseline, true
	case "stretch":
		return AlignItemsStretch, true
	default:
		return 0, false
	}
}

func parseAlignContent(s string) (AlignContent, bool) {
	switch s {
	case "start":
		return AlignContentStart, true
	case "end":
		return AlignContentEnd, true
	case "flex-start":
		return AlignContentFlexStart, true
	case "flex-end":
		return AlignContentFlexEnd, true
	case "center":
		return AlignContentCenter, true
	case "stretch":
		return AlignContentStretch, true
	case "space-between":
		return AlignContentSpaceBetween, true
	case "space-evenly":
		return AlignContentSpaceEvenly, true
	case "space-around":
		return AlignContentSpaceAround, true
	default:
		return 0, false
	}
}

func parseGridAutoFlow(s string) GridAutoFlow {
	switch s {
	case "column":
		return GridAutoFlowColumn
	case "row dense":
		return GridAutoFlowRowDense
	case "column dense":
		return GridAutoFlowColumnDense
	default:
		return GridAutoFlowRow
	}
}

// ---------------------------------------------------------------------------
// Dimension / LP / LPA parsers
// ---------------------------------------------------------------------------

// parseDimension parses a CSS dimension value like "100px", "50%", "auto".
func parseDimension(s string) Dimension {
	switch s {
	case "auto":
		return Dimension{Tag: DimensionAuto}
	case "min-content", "max-content":
		return Dimension{Tag: DimensionAuto}
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return DimensionFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return DimensionFromLength(float32(val))
}

// parseLPA parses a LengthPercentageAuto value.
func parseLPA(s string) LengthPercentageAuto {
	switch s {
	case "auto":
		return LengthPercentageAuto{Tag: LPAAuto}
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return LPAFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return LPAFromLength(float32(val))
}

// parseLP parses a LengthPercentage value.
func parseLP(s string) LengthPercentage {
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return LPFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return LPFromLength(float32(val))
}

// ---------------------------------------------------------------------------
// Grid Template Parsing
// ---------------------------------------------------------------------------

// parseGridTemplate parses a grid-template-rows or grid-template-columns value.
// Examples: "40px 40px 40px", "1fr 2fr", "repeat(3, 1fr)", "minmax(100px, 1fr)"
func parseGridTemplate(s string) []GridTemplateComponent {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return nil
	}

	var components []GridTemplateComponent
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
			components = append(components, GridTemplateSingleVal(tsf))
		} else if strings.HasPrefix(tok, "fit-content(") {
			tsf := parseFitContent(tok)
			components = append(components, GridTemplateSingleVal(tsf))
		} else {
			tsf := parseSingleTrack(tok)
			components = append(components, GridTemplateSingleVal(tsf))
		}
	}

	return components
}

// parseGridAutoTracks parses a grid-auto-rows or grid-auto-columns value.
// These are space-separated TrackSizingFunction values (no repeat allowed).
func parseGridAutoTracks(s string) []TrackSizingFunction {
	s = strings.TrimSpace(s)
	if s == "" || s == "none" {
		return nil
	}

	var tracks []TrackSizingFunction
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
func parseSingleTrack(s string) TrackSizingFunction {
	switch s {
	case "auto":
		return TrackSizingAuto
	case "min-content":
		return TrackSizingMinContent
	case "max-content":
		return TrackSizingMaxContent
	}
	if strings.HasSuffix(s, "fr") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "fr"), 32)
		return TrackFromFr(float32(val))
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return TrackFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return TrackFromLength(float32(val))
}

// parseMinmax parses "minmax(min, max)" into a TrackSizingFunction.
func parseMinmax(s string) TrackSizingFunction {
	// Strip "minmax(" and trailing ")"
	inner := s[7 : len(s)-1]
	parts := splitComma(inner)
	if len(parts) != 2 {
		panic(fmt.Sprintf("parseMinmax: expected 2 parts, got %d in %q", len(parts), s))
	}
	minPart := strings.TrimSpace(parts[0])
	maxPart := strings.TrimSpace(parts[1])
	return TrackSizingFunction{
		Min: parseMinTrackSizing(minPart),
		Max: parseMaxTrackSizing(maxPart),
	}
}

// parseFitContent parses "fit-content(200px)" or "fit-content(50%)" into a TrackSizingFunction.
func parseFitContent(s string) TrackSizingFunction {
	inner := s[12 : len(s)-1]
	inner = strings.TrimSpace(inner)
	return TrackFitContent(parseLP(inner))
}

// parseRepeat parses "repeat(3, 40px)" or "repeat(auto-fill, minmax(...))" etc.
func parseRepeat(s string) GridTemplateComponent {
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
	var tracks []TrackSizingFunction
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

	return GridTemplateRepeatVal(GridTemplateRepetition{
		Count:  count,
		Tracks: tracks,
	})
}

// parseRepetitionCount parses the count argument in repeat().
func parseRepetitionCount(s string) RepetitionCount {
	switch s {
	case "auto-fill":
		return RepetitionAutoFillVal
	case "auto-fit":
		return RepetitionAutoFitVal
	default:
		n, _ := strconv.Atoi(s)
		return RepetitionCountFromU16(uint16(n))
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
func parseMinTrackSizing(s string) MinTrackSizingFunction {
	switch s {
	case "auto":
		return MinTrackAutoVal
	case "min-content":
		return MinTrackMinContentVal
	case "max-content":
		return MinTrackMaxContentVal
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return MinTrackFromPercent(float32(val) / 100.0)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return MinTrackFromLength(float32(val))
}

// parseMaxTrackSizing parses a max track sizing function string.
func parseMaxTrackSizing(s string) MaxTrackSizingFunction {
	switch s {
	case "auto":
		return MaxTrackAutoVal
	case "min-content":
		return MaxTrackMinContentVal
	case "max-content":
		return MaxTrackMaxContentVal
	}
	if strings.HasSuffix(s, "fr") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "fr"), 32)
		return MaxTrackFromFr(float32(val))
	}
	if strings.HasSuffix(s, "%") {
		val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "%"), 32)
		return MaxTrackFromPercent(float32(val) / 100.0)
	}
	if strings.HasPrefix(s, "fit-content(") {
		inner := s[12 : len(s)-1]
		lp := parseLP(strings.TrimSpace(inner))
		return MaxTrackFitContentFromLP(lp)
	}
	val, _ := strconv.ParseFloat(strings.TrimSuffix(s, "px"), 32)
	return MaxTrackFromLength(float32(val))
}

// ---------------------------------------------------------------------------
// Grid Placement Parsing
// ---------------------------------------------------------------------------

// parseGridPlacement parses a grid placement value like "1", "span 2", "-1", "auto".
func parseGridPlacement(s string) GridPlacement {
	s = strings.TrimSpace(s)
	switch s {
	case "", "auto":
		return GridPlacementAutoVal
	}
	if strings.HasPrefix(s, "span ") {
		rest := strings.TrimSpace(s[5:])
		n, _ := strconv.Atoi(rest)
		return GridPlacementFromSpan(uint16(n))
	}
	n, err := strconv.Atoi(s)
	if err != nil {
		return GridPlacementAutoVal
	}
	return GridPlacementFromLine(int16(n))
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
func buildTree(tree *TaffyTree, node *testNode) NodeId {
	style := buildStyle(node.Attrs)

	if len(node.Children) > 0 {
		// Internal node: build children first
		childIds := make([]NodeId, len(node.Children))
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
		knownDimensions Size[Option[float32]],
		parentSize Size[Option[float32]],
		availableSpace Size[AvailableSpace],
		sizingMode SizingMode,
	) Size[float32] {
		// If both dimensions are known, return them directly
		if knownDimensions.Width.IsSome && knownDimensions.Height.IsSome {
			return Size[float32]{
				Width:  knownDimensions.Width.Value,
				Height: knownDimensions.Height.Value,
			}
		}

		lines := strings.Split(text, string(zwsp))

		if len(lines) == 0 {
			return SizeZeroF32
		}

		// Determine inline and block axes based on writing mode
		var inlineAxis, blockAxis AbsoluteAxis
		if wm == writingModeHorizontal {
			inlineAxis = AbsoluteAxisHorizontal
			blockAxis = AbsoluteAxisVertical
		} else {
			inlineAxis = AbsoluteAxisVertical
			blockAxis = AbsoluteAxisHorizontal
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
			case AvailableSpaceMinContent:
				inlineSize = float32(minLineLen) * ahemCharWidth
			case AvailableSpaceMaxContent:
				inlineSize = float32(maxLineLen) * ahemCharWidth
			case AvailableSpaceDefinite:
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
			return Size[float32]{Width: inlineSize, Height: blockSize}
		}
		return Size[float32]{Width: blockSize, Height: inlineSize}
	}
}

// Helper to get AvailableSpace by absolute axis.
func sizeAvailableGetAbs(s Size[AvailableSpace], axis AbsoluteAxis) AvailableSpace {
	if axis == AbsoluteAxisHorizontal {
		return s.Width
	}
	return s.Height
}

// Helper to get Option[float32] by absolute axis.
func sizeGetAbs(s Size[Option[float32]], axis AbsoluteAxis) Option[float32] {
	if axis == AbsoluteAxisHorizontal {
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
	knownDimensions Size[Option[float32]],
	availableSpace Size[AvailableSpace],
	nodeId NodeId,
	context *NodeContext,
	style *Style,
) Size[float32] {
	if knownDimensions.Width.IsSome && knownDimensions.Height.IsSome {
		return Size[float32]{
			Width:  knownDimensions.Width.Value,
			Height: knownDimensions.Height.Value,
		}
	}

	if context == nil || *context == nil {
		return Size[float32]{
			Width:  knownDimensions.Width.UnwrapOr(0),
			Height: knownDimensions.Height.UnwrapOr(0),
		}
	}

	computeSize := (*context)(knownDimensions, SizeOptionNone, availableSpace, SizingModeInherentSize)

	return Size[float32]{
		Width:  knownDimensions.Width.UnwrapOr(computeSize.Width),
		Height: knownDimensions.Height.UnwrapOr(computeSize.Height),
	}
}

// ---------------------------------------------------------------------------
// Layout Assertion
// ---------------------------------------------------------------------------

// assertLayoutMatches recursively checks that the computed layout matches expectations.
func assertLayoutMatches(tree *TaffyTree, nodeId NodeId, expected *expectedNode) {
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
