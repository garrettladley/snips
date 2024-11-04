- [ X ] Determine package name to be generated
- [ ] Determine language to be used in lexer
```go
lexer := lexers.Match("foo.go")
if lexer == nil {
lexer = lexers.Fallback
}
chroma.Coalesce(lexer)
```
- [ ] Drill down style
- [ ] Flag for with version
- [ ] Flag for with timestamp
- [ ] Do we need lazy?
- [ ] Do we need the post generation loop? If so, we should add back the cmd flag
