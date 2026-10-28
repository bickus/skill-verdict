// SPDX-License-Identifier: Apache-2.0
// Copyright 2026 skill-verdict authors

package code

import "github.com/bickus/skill-verdict/pkg/internal/syntax"

func set(names ...string) map[string]bool {
	out := make(map[string]bool, len(names))
	for _, n := range names {
		out[n] = true
	}
	return out
}

var pythonSpec = &spec{
	language:      syntax.Python,
	comprehension: "for_in_clause",
	funcs:         set("function_definition", "lambda"),
	blocks:        set("module", "block", "elif_clause", "else_clause", "except_clause", "except_group_clause", "finally_clause", "with_clause", "with_item", "case_clause"),
	simple: set("expression_statement", "return_statement", "import_statement", "import_from_statement", "future_import_statement",
		"assert_statement", "raise_statement", "delete_statement", "global_statement", "nonlocal_statement", "print_statement", "exec_statement"),
	assigns:    map[string][2]string{"assignment": {"left", "right"}, "augmented_assignment": {"left", "right"}},
	loops:      set("for_statement"),
	calls:      map[string]string{"call": "function"},
	members:    map[string]member{"attribute": {object: "object", property: "attribute"}, "subscript": {object: "value", property: "subscript", index: true}},
	idents:     set("identifier"),
	strings:    set("string", "concatenated_string"),
	fragments:  set("string_content", "escape_sequence"),
	delimiters: set("string_start", "string_end"),
	objects:    set(),
	skipped:    set("type"),
	imports:    set("import_statement", "import_from_statement", "future_import_statement"),
	selves:     set("self", "cls"),
	comment:    "comment",
	keyword:    "keyword_argument",
	pair:       "pair",
	returns:    "return_statement",
	alias:      "as_pattern",
	docstrings: true,
}

func scriptSpec(language syntax.Language) *spec {
	return &spec{
		language: language,
		funcs:    set("function_declaration", "generator_function_declaration", "function_expression", "generator_function", "arrow_function", "method_definition"),
		blocks:   set("program", "statement_block", "else_clause", "class_body", "switch_body", "switch_case", "switch_default", "catch_clause", "finally_clause"),
		simple: set("expression_statement", "return_statement", "lexical_declaration", "variable_declaration", "import_statement",
			"throw_statement", "break_statement", "continue_statement"),
		assigns: map[string][2]string{"variable_declarator": {"name", "value"}, "assignment_expression": {"left", "right"},
			"augmented_assignment_expression": {"left", "right"}},
		loops:     set("for_in_statement"),
		calls:     map[string]string{"call_expression": "function", "new_expression": "constructor"},
		members:   map[string]member{"member_expression": {object: "object", property: "property"}, "subscript_expression": {object: "object", property: "index", index: true}},
		idents:    set("identifier"),
		strings:   set("string", "template_string"),
		fragments: set("string_fragment", "escape_sequence"),
		objects:   set("object", "object_pattern"),
		skipped: set("type_annotation", "type_arguments", "type_parameters", "predefined_type", "type_alias_declaration", "interface_declaration",
			"ambient_declaration"),
		imports:   set("import_statement"),
		selves:    set("this"),
		comment:   "comment",
		pair:      "pair",
		shorthand: "shorthand_property_identifier",
		returns:   "return_statement",
	}
}

var specs = map[Lang]*spec{
	Python:     pythonSpec,
	JavaScript: scriptSpec(syntax.JavaScript),
	TypeScript: scriptSpec(syntax.TypeScript),
	TSX:        scriptSpec(syntax.TSX),
}
