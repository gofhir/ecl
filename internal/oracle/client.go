// Package oracle implements a differential test harness: it evaluates an ECL
// expression twice — once with this library over a DataProvider backed by a FHIR
// terminology server, and once by asking that same server to evaluate the whole
// expression itself — and compares the two concept sets.
//
// # Why this exists
//
// Every semantic expectation in this repository was written by its authors. The
// specification gives prose and examples without expected results, so there is no
// corpus that says "this expression over SNOMED International returns these
// concepts". A misreading of the spec therefore produces a passing test, and that
// is not hypothetical: several bugs fixed in v1.2.0 had a green test asserting the
// wrong behavior.
//
// A reference implementation is the only available judge. The server answers the
// primitive questions (what are the descendants of X, what are the targets of
// attribute A on the concepts in S) and also the whole expression; this library
// only supplies the composition — refinement, cardinality, negation, grouping,
// set algebra — which is where every bug this project has found actually lived.
// A divergence is therefore a divergence in composition.
//
// # What it does not prove
//
// The server is another implementation, not the specification. A divergence can
// mean it is wrong, or that the text genuinely admits two readings. Triage the
// finding, do not reflexively change this library to match.
//
// # Cost
//
// It talks to a public server over the network. Nothing here runs during a normal
// `go test ./...`: the test skips unless ECL_ORACLE_URL is set. Expressions are
// bounded so a single case costs a handful of requests, not thousands.
package oracle

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

// pageSize is how many codes to ask for per $expand page. Large enough that the
// bounded expressions in the corpus fit in one page.
const pageSize = 1000

// batchSize is how many $lookup calls to put in one FHIR batch Bundle. Attribute
// relationships are only available per concept, so without batching a focus of
// 300 concepts would be 300 round trips.
const batchSize = 50

// maxExpansion caps how large an intermediate concept set may get.
//
// It is a guard against a corpus expression whose focus is most of SNOMED CT: the
// per-concept relationship lookups would then be tens of thousands of requests
// against a public server. Exceeding it is a corpus problem — bound the expression — not a
// transient failure, so it reports as an error rather than a retry.
const maxExpansion = 4000

// Client is a minimal FHIR terminology client. It speaks only the two operations
// this harness needs.
type Client struct {
	baseURL string
	http    *http.Client

	// expandCache and propertyCache make repeated primitives free within a run.
	// The evaluator asks for the same descendants many times over a corpus, and a
	// public server should not be asked twice for an immutable answer.
	expandCache   map[string][]string
	propertyCache map[string]ConceptProperties

	// Requests counts HTTP round trips, so a test can report the cost it imposed.
	Requests int
}

// NewClient returns a Client for a FHIR R4 terminology endpoint, for example
// https://r4.ontoserver.csiro.au/fhir.
func NewClient(baseURL string) *Client {
	return &Client{
		baseURL:       strings.TrimSuffix(baseURL, "/"),
		http:          &http.Client{Timeout: 90 * time.Second},
		expandCache:   map[string][]string{},
		propertyCache: map[string]ConceptProperties{},
	}
}

// ExpandECL evaluates an ECL expression on the server and returns the concept
// IDs, following pages until the expansion is complete.
//
// This is both how the provider answers primitives and how the oracle answers the
// whole expression. That is deliberate: the two sides of the comparison reach the
// same data through the same operation, so a divergence cannot be blamed on one
// side seeing a different SNOMED CT edition.
func (c *Client) ExpandECL(ctx context.Context, expr string) ([]string, error) {
	if cached, ok := c.expandCache[expr]; ok {
		return cached, nil
	}

	var codes []string
	for offset := 0; ; offset += pageSize {
		page, total, err := c.expandPage(ctx, expr, offset)
		if err != nil {
			return nil, err
		}
		if total > maxExpansion {
			return nil, fmt.Errorf("%w: %q expands to %d concepts (cap %d) — bound the expression",
				ErrTooLarge, expr, total, maxExpansion)
		}
		codes = append(codes, page...)
		if len(page) == 0 || len(codes) >= total {
			break
		}
	}

	c.expandCache[expr] = codes
	return codes, nil
}

// expandPage requests one page of a ValueSet/$expand.
//
// It POSTs rather than GETs: the provider translates a set of concept IDs into an
// ECL disjunction, which for a few hundred concepts is far past any sane URL
// length limit.
func (c *Client) expandPage(ctx context.Context, expr string, offset int) (codes []string, total int, err error) {
	body, err := json.Marshal(parameters{
		ResourceType: "Parameters",
		Parameter: []parameter{
			{Name: "url", ValueURI: "http://snomed.info/sct?fhir_vs=ecl/" + expr},
			{Name: "count", ValueInteger: ptr(pageSize)},
			{Name: "offset", ValueInteger: ptr(offset)},
			{Name: "includeDesignations", ValueBoolean: ptr(false)},
		},
	})
	if err != nil {
		return nil, 0, err
	}

	var out struct {
		ResourceType string `json:"resourceType"`
		Expansion    struct {
			Total    int `json:"total"`
			Contains []struct {
				Code string `json:"code"`
			} `json:"contains"`
		} `json:"expansion"`
		Issue []issue `json:"issue"`
	}
	if err := c.post(ctx, "/ValueSet/$expand", body, &out); err != nil {
		return nil, 0, err
	}
	if out.ResourceType == "OperationOutcome" {
		return nil, 0, fmt.Errorf("%w: expanding %q: %s", ErrServerRejected, expr, diagnostics(out.Issue))
	}

	codes = make([]string, 0, len(out.Expansion.Contains))
	for _, c := range out.Expansion.Contains {
		codes = append(codes, c.Code)
	}
	return codes, out.Expansion.Total, nil
}

// Attr is one raw attribute relationship of a concept, as the server reports it.
type Attr struct {
	TypeID string

	// ValueCode is the target concept, empty when Concrete is set.
	ValueCode string

	// Concrete holds a non-concept value: a decimal, integer, string or boolean.
	Concrete *ConcreteLiteral
}

// ConcreteLiteral is a concrete attribute value, kept as text because the
// evaluator parses it itself.
type ConcreteLiteral struct {
	Kind  string // "integer", "decimal", "string" or "boolean"
	Value string
}

// ConceptProperties is a concept's attribute relationships, split the way ECL
// reads them: an ungrouped set plus zero or more relationship groups.
type ConceptProperties struct {
	Ungrouped []Attr
	Groups    [][]Attr
}

// Properties returns the raw attribute relationships of each concept, grouped.
//
// This uses $lookup with property=*, which returns the concept's own inferred
// relationships: ungrouped attributes as top-level properties keyed by attribute
// SCTID, and each relationship group as a property with code 609096000 whose
// subproperties are the attributes of that group.
//
// The first version of this harness derived the same information from the long
// normal form instead, because $lookup's default response does not include
// relationships and it was not obvious that asking for them was possible. That was
// wrong in a way worth recording: the long normal form REPLACES an attribute's
// target with the target's own definition, so the raw target id is not recoverable
// from it. Measured on 17531000119105, whose normal form renders "due to" as
// (63739005|Coronary occlusion|:{...}) while the actual relationship target is
// 123641001|Left coronary artery occlusion|. Reading the focus concept of the
// nested expression as the target would have produced a provider that quietly fed
// this library different facts than the server used for the same query, and every
// resulting divergence would have looked like a defect in the evaluator.
//
// Requests are sent in FHIR batch Bundles: a focus of 300 concepts is six round
// trips rather than 300, which is also what the ecl.BatchPropertiesProvider
// capability exists for.
//
// A concept the server does not know is absent from the result rather than an
// error, mirroring the DataProvider convention for missing concepts.
func (c *Client) Properties(ctx context.Context, conceptIDs []string) (map[string]ConceptProperties, error) {
	out := make(map[string]ConceptProperties, len(conceptIDs))

	var missing []string
	for _, id := range conceptIDs {
		if props, ok := c.propertyCache[id]; ok {
			out[id] = props
			continue
		}
		missing = append(missing, id)
	}

	for start := 0; start < len(missing); start += batchSize {
		end := min(start+batchSize, len(missing))
		chunk := missing[start:end]

		entries := make([]bundleEntry, 0, len(chunk))
		for _, id := range chunk {
			entries = append(entries, bundleEntry{Request: bundleRequest{
				Method: "GET",
				URL: "CodeSystem/$lookup?system=" + url.QueryEscape("http://snomed.info/sct") +
					"&code=" + url.QueryEscape(id) + "&property=" + url.QueryEscape("*"),
			}})
		}
		body, err := json.Marshal(bundle{ResourceType: "Bundle", Type: "batch", Entry: entries})
		if err != nil {
			return nil, err
		}

		var resp struct {
			Entry []struct {
				Resource struct {
					Parameter []lookupParameter `json:"parameter"`
				} `json:"resource"`
			} `json:"entry"`
		}
		if err := c.post(ctx, "", body, &resp); err != nil {
			return nil, err
		}
		if len(resp.Entry) != len(chunk) {
			return nil, fmt.Errorf("%w: batch of %d returned %d entries",
				ErrServerRejected, len(chunk), len(resp.Entry))
		}

		// Entries come back in request order, which is what pairs a response with
		// its concept: a $lookup result carries the code but a 404 entry does not.
		for i, e := range resp.Entry {
			props := propertiesOf(e.Resource.Parameter)
			c.propertyCache[chunk[i]] = props
			out[chunk[i]] = props
		}
	}

	return out, nil
}

// metadataProperties are the property codes that describe the concept rather than
// relate it to another concept. Everything else at the top level is an ungrouped
// attribute, keyed by its attribute SCTID.
var metadataProperties = map[string]bool{
	"inactive": true, "parent": true, "child": true,
	"sufficientlyDefined": true, "effectiveTime": true, "moduleId": true,
	"definitionStatusId": true, "normalForm": true, "normalFormTerse": true,
	"designation": true, "inactivationIndicator": true, "moduleName": true,
	roleGroupProperty: true,
}

// roleGroupProperty is the SCTID FHIR uses to wrap one relationship group.
const roleGroupProperty = "609096000"

func propertiesOf(params []lookupParameter) ConceptProperties {
	var props ConceptProperties
	for _, p := range params {
		if p.Name != "property" {
			continue
		}
		code, value := "", (*Attr)(nil)
		var subs []Attr
		for _, part := range p.Part {
			switch part.Name {
			case "code":
				code = part.ValueCode + part.ValueString
			case "value":
				value = attrValue("", part)
			case "subproperty":
				if a := subAttr(part.Part); a != nil {
					subs = append(subs, *a)
				}
			}
		}

		switch {
		case code == roleGroupProperty:
			// An empty group would mean the server reported a group with no
			// attribute in it; keeping it would create a phantom group that
			// group cardinality would then count.
			if len(subs) > 0 {
				props.Groups = append(props.Groups, subs)
			}
		case metadataProperties[code] || value == nil:
			// Concept metadata, not a relationship.
		default:
			value.TypeID = code
			props.Ungrouped = append(props.Ungrouped, *value)
		}
	}
	return props
}

func subAttr(parts []lookupPart) *Attr {
	code := ""
	var value *Attr
	for _, part := range parts {
		switch part.Name {
		case "code":
			code = part.ValueCode + part.ValueString
		case "value":
			value = attrValue(code, part)
		}
	}
	if code == "" || value == nil {
		return nil
	}
	value.TypeID = code
	return value
}

// attrValue reads whichever value[x] the server used. A concept-valued attribute
// arrives as valueCode; the concrete types arrive as themselves.
func attrValue(typeID string, part lookupPart) *Attr {
	switch {
	case part.ValueCode != "":
		return &Attr{TypeID: typeID, ValueCode: part.ValueCode}
	case part.ValueDecimal != nil:
		return &Attr{TypeID: typeID, Concrete: &ConcreteLiteral{
			Kind: "decimal", Value: strconv.FormatFloat(*part.ValueDecimal, 'f', -1, 64)}}
	case part.ValueInteger != nil:
		return &Attr{TypeID: typeID, Concrete: &ConcreteLiteral{
			Kind: "integer", Value: strconv.Itoa(*part.ValueInteger)}}
	case part.ValueBoolean != nil:
		return &Attr{TypeID: typeID, Concrete: &ConcreteLiteral{
			Kind: "boolean", Value: strconv.FormatBool(*part.ValueBoolean)}}
	case part.ValueString != "":
		return &Attr{TypeID: typeID, Concrete: &ConcreteLiteral{Kind: "string", Value: part.ValueString}}
	}
	return nil
}

// post sends one request and decodes the JSON response. An empty path posts to
// the base URL itself, which is how a batch Bundle is submitted.
func (c *Client) post(ctx context.Context, path string, body []byte, into any) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+path, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/fhir+json")
	req.Header.Set("Accept", "application/fhir+json")

	c.Requests++
	resp, err := c.http.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %w", ErrUnreachable, err)
	}
	defer func() { _ = resp.Body.Close() }()

	// A batch Bundle answers 200 with per-entry statuses, and $expand answers 200
	// or an OperationOutcome with a 4xx. Anything else is not something to parse.
	if resp.StatusCode >= 500 {
		return fmt.Errorf("%w: HTTP %d from %s", ErrUnreachable, resp.StatusCode, c.baseURL+path)
	}
	if err := json.NewDecoder(resp.Body).Decode(into); err != nil {
		return fmt.Errorf("%w: decoding HTTP %d: %w", ErrServerRejected, resp.StatusCode, err)
	}
	return nil
}

func diagnostics(issues []issue) string {
	if len(issues) == 0 {
		return "no diagnostics"
	}
	return issues[0].Diagnostics
}

func ptr[T any](v T) *T { return &v }

// ── FHIR wire types, trimmed to the fields this harness reads ────────────────.

type parameters struct {
	ResourceType string      `json:"resourceType"`
	Parameter    []parameter `json:"parameter"`
}

type parameter struct {
	Name         string `json:"name"`
	ValueURI     string `json:"valueUri,omitempty"`
	ValueInteger *int   `json:"valueInteger,omitempty"`
	ValueBoolean *bool  `json:"valueBoolean,omitempty"`
}

type bundle struct {
	ResourceType string        `json:"resourceType"`
	Type         string        `json:"type"`
	Entry        []bundleEntry `json:"entry"`
}

type bundleEntry struct {
	Request bundleRequest `json:"request"`
}

type bundleRequest struct {
	Method string `json:"method"`
	URL    string `json:"url"`
}

type issue struct {
	Diagnostics string `json:"diagnostics"`
}

// lookupParameter and lookupPart model the recursive Parameters resource a
// $lookup returns. A part carries at most one value[x]; pointers distinguish
// "absent" from a zero decimal or a false boolean, which for a concrete attribute
// value is a real distinction.
type lookupParameter struct {
	Name string       `json:"name"`
	Part []lookupPart `json:"part"`
}

type lookupPart struct {
	Name         string       `json:"name"`
	ValueCode    string       `json:"valueCode"`
	ValueString  string       `json:"valueString"`
	ValueDecimal *float64     `json:"valueDecimal"`
	ValueInteger *int         `json:"valueInteger"`
	ValueBoolean *bool        `json:"valueBoolean"`
	Part         []lookupPart `json:"part"`
}
