#include <stdint.h>
#include <stdlib.h>
#include <string.h>

#include "tree_sitter/api.h"

const TSLanguage *tree_sitter_python(void);
const TSLanguage *tree_sitter_javascript(void);
const TSLanguage *tree_sitter_typescript(void);
const TSLanguage *tree_sitter_tsx(void);
const TSLanguage *tree_sitter_powershell(void);

#define COLUMNS 5

static uint32_t out_size;

static const TSLanguage *pick(uint32_t id) {
  switch (id) {
    case 1: return tree_sitter_python();
    case 2: return tree_sitter_javascript();
    case 3: return tree_sitter_typescript();
    case 4: return tree_sitter_tsx();
    case 5: return tree_sitter_powershell();
  }
  return NULL;
}

__attribute__((export_name("sv_alloc"))) void *sv_alloc(uint32_t size) { return malloc(size); }

__attribute__((export_name("sv_free"))) void sv_free(void *ptr) { free(ptr); }

__attribute__((export_name("sv_size"))) uint32_t sv_size(void) { return out_size; }

static uint32_t *flatten(TSTree *tree) {
  uint32_t cap = 4096;
  uint32_t count = 0;
  uint32_t depth = 0;
  uint32_t *rows = malloc(cap * COLUMNS * sizeof(uint32_t));
  TSTreeCursor cursor = ts_tree_cursor_new(ts_tree_root_node(tree));
  int more = rows != NULL;
  while (more) {
    TSNode node = ts_tree_cursor_current_node(&cursor);
    if (count == cap) {
      cap *= 2;
      uint32_t *grown = realloc(rows, cap * COLUMNS * sizeof(uint32_t));
      if (grown == NULL) break;
      rows = grown;
    }
    uint32_t *row = rows + count * COLUMNS;
    row[0] = ts_node_symbol(node);
    row[1] = ts_node_start_byte(node);
    row[2] = ts_node_end_byte(node);
    row[3] = depth << 3 | (uint32_t)ts_node_is_error(node) << 2 | (uint32_t)ts_node_is_missing(node) << 1 | (uint32_t)ts_node_is_named(node);
    row[4] = ts_tree_cursor_current_field_id(&cursor);
    count++;
    if (ts_tree_cursor_goto_first_child(&cursor)) {
      depth++;
      continue;
    }
    while (!ts_tree_cursor_goto_next_sibling(&cursor)) {
      if (!ts_tree_cursor_goto_parent(&cursor)) {
        more = 0;
        break;
      }
      depth--;
    }
  }
  ts_tree_cursor_delete(&cursor);
  out_size = count * COLUMNS * sizeof(uint32_t);
  return rows;
}

__attribute__((export_name("sv_parse"))) uint32_t *sv_parse(uint32_t lang, const char *source, uint32_t length) {
  out_size = 0;
  const TSLanguage *language = pick(lang);
  TSParser *parser = ts_parser_new();
  if (language == NULL || !ts_parser_set_language(parser, language)) {
    ts_parser_delete(parser);
    return NULL;
  }
  TSTree *tree = ts_parser_parse_string(parser, NULL, source, length);
  ts_parser_delete(parser);
  if (tree == NULL) return NULL;
  uint32_t *rows = flatten(tree);
  ts_tree_delete(tree);
  return rows;
}

static uint32_t append(char *out, uint32_t at, const char *name) {
  if (name == NULL) name = "";
  uint32_t size = strlen(name) + 1;
  if (out != NULL) memcpy(out + at, name, size);
  return at + size;
}

static uint32_t names(const TSLanguage *language, char *out) {
  uint32_t symbols = ts_language_symbol_count(language);
  uint32_t fields = ts_language_field_count(language);
  uint32_t at = 2 * sizeof(uint32_t);
  if (out != NULL) {
    memcpy(out, &symbols, sizeof(uint32_t));
    memcpy(out + sizeof(uint32_t), &fields, sizeof(uint32_t));
  }
  for (uint32_t i = 0; i < symbols; i++) at = append(out, at, ts_language_symbol_name(language, i));
  for (uint32_t i = 1; i <= fields; i++) at = append(out, at, ts_language_field_name_for_id(language, i));
  return at;
}

__attribute__((export_name("sv_names"))) char *sv_names(uint32_t lang) {
  out_size = 0;
  const TSLanguage *language = pick(lang);
  if (language == NULL) return NULL;
  uint32_t size = names(language, NULL);
  char *out = malloc(size);
  if (out == NULL) return NULL;
  names(language, out);
  out_size = size;
  return out;
}
