# JFace factories

Example: [`examples/jface`](../examples/jface).

Package `github.com/haiodo/gowt/jface` is a slice of Eclipse JFace, translated from `eclipse.platform.ui`. It works on `swt` types, not on the `gowt` facade, and builds on all three OSes from one source.

## What is there

| Group | Names | Use |
|---|---|---|
| Widget factories | `WidgetFactoryButton`, `WidgetFactoryLabel`, `WidgetFactoryText`, `WidgetFactoryGroup`, `WidgetFactoryComposite`, `WidgetFactoryShell`, `WidgetFactorySash`, `WidgetFactorySashForm`, `WidgetFactorySpinner`, `WidgetFactoryDateTime`, `WidgetFactoryTable`, `WidgetFactoryTableColumn`, `WidgetFactoryTree`, `WidgetFactoryTreeColumn`, `LinkFactoryNewLink` | A fluent chain ends in `Create(parent)` and returns the plain `swt` widget: `jface.WidgetFactoryButton(swt.PUSH).Text("OK").OnSelect(f).Create(parent)` |
| Layout factories | `GridLayoutFactory` (`FillDefaults`, `SwtDefaults`), `GridDataFactory`, `FillLayoutFactory`, `RowLayoutFactory`, `RowDataFactory` | Build or apply layouts and layout data: `jface.GridLayoutFactoryFillDefaults().NumColumns(2).ApplyTo(composite)` |
| Column layouts | `TableColumnLayout`, `TreeColumnLayout`, `ColumnWeightData`, `ColumnPixelData` | Table and tree columns sized by weight or pixels |
| Helpers | `LayoutConstants*`, `PixelConverter`, `Geometry*`, `IDialogConstants*` | Standard margins, dialog units, rectangle arithmetic, dialog button ids |

`go doc github.com/haiodo/gowt/jface` lists every name. Parameters that SWT declares as `Control` or `Composite` are the `swt.ControlLike` and `swt.CompositeLike` interfaces here.

## What is not there

- No viewers: no `TableViewer`, `TreeViewer`, label or content providers.
- No dialogs, wizards, preferences, actions or resource registries. `Dialog` and `JFaceResources` are two small stand-ins that the layout code needs (`jface/dialogs_manual.go`).

## Using it with gowt containers

`Window`, `Panel`, `Group`, `Split` and `CoolBar` have `AsComposite() *swt.Composite`, which has the same type on all of them. It is the parent a factory wants:

```go
package main

import (
	"log"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/jface"
	"github.com/haiodo/gowt/swt"
)

func main() {
	err := gowt.Run(func(app *gowt.App) {
		w := app.Window("JFace factories")
		jface.GridLayoutFactorySwtDefaults().NumColumns(2).ApplyTo(w.AsComposite())

		fill := func() *swt.GridData { return jface.GridDataFactoryFillDefaults().Grab(true, false).Create() }
		jface.WidgetFactoryLabel(swt.NONE).Text("Name:").Create(w.AsComposite())
		name := jface.WidgetFactoryText(swt.BORDER).Message("Full name").LayoutData(fill()).Create(w.AsComposite())

		status := w.Label("", gowt.Cell(gowt.GridCell{SpanX: 2, Align: gowt.AlignFill, GrowX: true}))
		jface.WidgetFactoryButton(swt.PUSH).Text("Greet").
			LayoutData(jface.GridDataFactoryCreate(swt.END).Span(2, 1).Create()).
			OnSelect(func(e *swt.SelectionEvent) { status.SetText("Hello, " + name.GetText()) }).
			Create(w.AsComposite())

		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}
```

Points to know:

- A factory returns an `*swt.Button`, `*swt.Text` and so on, not a gowt wrapper. Call its `swt` methods (`GetText`, `SetText`). There is no constructor that wraps an existing `swt` widget into a gowt one.
- A JFace layout and a gowt layout are both SWT layouts. Applying a JFace layout to a gowt container is fine; `SetLayout` on it afterwards replaces it. `gowt.Cell(GridCell{...})` produces the same `GridData` that `GridDataFactory` does, so both kinds of child can sit in one grid.
- Apply the layout through `AsComposite()`, not through `gowt`'s `SetLayout`, when the layout is a `jface` one: `SetLayout` takes a gowt `Layout` value.
- The same rules as for the rest of gowt: touch widgets only on the UI thread ([events.md](events.md)).

`cmd/jfacedemo` builds a whole form with factories only (no `gowt`), on `swt` directly.
