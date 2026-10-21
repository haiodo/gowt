package swt

import "github.com/haiodo/gowt/internal/gtk"

// visibleRowCount counts the rows below parent that are shown: those whose ancestors are all expanded.
// After a node is collapsed and expanded again GtkTreeView still reports the collapsed height, so
// computeSize takes the larger of its height and this count.
func (this *Tree) visibleRowCount(parent int64) int32 {
	iter := gtk.OSG_malloc(int64(gtk.GTKGtkTreeIter_sizeof()))
	defer gtk.OSG_free(iter)
	var rows int32
	for ok := gtk.GTKGtk_tree_model_iter_children(this.modelHandle, iter, parent); ok; ok = gtk.GTKGtk_tree_model_iter_next(this.modelHandle, iter) {
		rows++
		path := gtk.GTKGtk_tree_model_get_path(this.modelHandle, iter)
		if gtk.GTKGtk_tree_view_row_expanded(this.Handle, path) {
			rows += this.visibleRowCount(iter)
		}
		gtk.GTKGtk_tree_path_free(path)
	}
	return rows
}
