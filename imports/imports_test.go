package imports_test

import (
	"os"

	. "github.com/philandstuff/dhall-golang/ast"
	. "github.com/philandstuff/dhall-golang/imports"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
)

func expectResolves(input, expected Expr) {
	os.Setenv("FOO", "abcd")
	actual, err := Load(input)

	Expect(err).ToNot(HaveOccurred())
	Expect(actual).To(Equal(expected))
}

var importFooAsText = Embed(MakeEnvVarImport("FOO", RawText))
var resolvedFooAsText = TextLit{Suffix: "abcd"}

var _ = Describe("Import resolution", func() {
	It("Leaves a literal expression untouched", func() {
		actual, err := Load(NaturalLit(3))

		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(NaturalLit(3)))
	})
	It("Resolves an `as Text` import", func() {
		os.Setenv("FOO", "abcd")
		actual, err := Load(importFooAsText)

		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(resolvedFooAsText))
	})
	It("Resolves and parses code import", func() {
		os.Setenv("FOO", "3 : Natural")
		actual, err := Load(Embed(MakeEnvVarImport("FOO", Code)))

		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(Annot{Expr: NaturalLit(3), Annotation: Natural}))
	})
	It("Performs import chaining", func() {
		os.Setenv("CHAIN1", "env:CHAIN2")
		os.Setenv("CHAIN2", "2 + 2")
		actual, err := Load(Embed(MakeEnvVarImport("CHAIN1", Code)))

		Expect(err).ToNot(HaveOccurred())
		Expect(actual).To(Equal(NaturalPlus{NaturalLit(2), NaturalLit(2)}))
	})
	It("Rejects import cycles", func() {
		os.Setenv("CYCLE", "env:CYCLE")
		_, err := Load(Embed(MakeEnvVarImport("CYCLE", Code)))

		Expect(err).To(HaveOccurred())
	})
	It("Fails to resolve a code import with free variables", func() {
		os.Setenv("FOO", "x")
		_, err := Load(Embed(MakeEnvVarImport("FOO", Code)))

		Expect(err).To(HaveOccurred())
	})
	DescribeTable("Subexpressions to resolve", expectResolves,
		Entry("Simple import", importFooAsText, resolvedFooAsText),
		Entry("Import within lambda type",
			&LambdaExpr{Type: importFooAsText},
			&LambdaExpr{Type: resolvedFooAsText},
		),
		Entry("Import within lambda body",
			&LambdaExpr{Body: importFooAsText},
			&LambdaExpr{Body: resolvedFooAsText},
		),
		Entry("Import within pi type",
			&Pi{Type: importFooAsText},
			&Pi{Type: resolvedFooAsText},
		),
		Entry("Import within pi body",
			&Pi{Body: importFooAsText},
			&Pi{Body: resolvedFooAsText},
		),
		Entry("Import within app fn",
			&App{Fn: importFooAsText},
			&App{Fn: resolvedFooAsText},
		),
		Entry("Import within app arg",
			&App{Arg: importFooAsText},
			&App{Arg: resolvedFooAsText},
		),
		Entry("Import within let binding value",
			MakeLet(Natural, Binding{Value: importFooAsText}),
			MakeLet(Natural, Binding{Value: resolvedFooAsText}),
		),
		Entry("Import within let body",
			MakeLet(importFooAsText, Binding{}),
			MakeLet(resolvedFooAsText, Binding{}),
		),
		Entry("Import within annotated expression",
			Annot{importFooAsText, Text},
			Annot{resolvedFooAsText, Text},
		),
		Entry("Import within annotation",
			// these don't typecheck but we're just
			// checking the imports here
			Annot{Natural, importFooAsText},
			Annot{Natural, resolvedFooAsText},
		),
		Entry("Import within TextLit",
			TextLit{
				Chunks: []Chunk{
					Chunk{
						Prefix: "foo",
						Expr:   importFooAsText,
					}},
				Suffix: "baz",
			},
			TextLit{
				Chunks: []Chunk{
					Chunk{
						Prefix: "foo",
						Expr:   resolvedFooAsText,
					},
				},
				Suffix: "baz",
			},
		),
		Entry("Import within if condition",
			BoolIf{Cond: importFooAsText},
			BoolIf{Cond: resolvedFooAsText},
		),
		Entry("Import within if true branch",
			BoolIf{T: importFooAsText},
			BoolIf{T: resolvedFooAsText},
		),
		Entry("Import within if false branch",
			BoolIf{F: importFooAsText},
			BoolIf{F: resolvedFooAsText},
		),
		Entry("Import within natural plus (left side)",
			NaturalPlus{L: importFooAsText},
			NaturalPlus{L: resolvedFooAsText},
		),
		Entry("Import within natural plus (right side)",
			NaturalPlus{R: importFooAsText},
			NaturalPlus{R: resolvedFooAsText},
		),
		Entry("Import within natural times (left side)",
			NaturalTimes{L: importFooAsText},
			NaturalTimes{L: resolvedFooAsText},
		),
		Entry("Import within natural times (right side)",
			NaturalTimes{R: importFooAsText},
			NaturalTimes{R: resolvedFooAsText},
		),
		Entry("Import within empty list type",
			EmptyList{Type: importFooAsText},
			EmptyList{Type: resolvedFooAsText},
		),
		Entry("Import within list",
			MakeList(importFooAsText),
			MakeList(resolvedFooAsText),
		),
		Entry("Import within record type",
			Record(map[string]Expr{"foo": importFooAsText}),
			Record(map[string]Expr{"foo": resolvedFooAsText}),
		),
		Entry("Import within record literal",
			RecordLit(map[string]Expr{"foo": importFooAsText}),
			RecordLit(map[string]Expr{"foo": resolvedFooAsText}),
		),
		Entry("Import within field extract",
			Field{Record: importFooAsText},
			Field{Record: resolvedFooAsText},
		),
	)
})
