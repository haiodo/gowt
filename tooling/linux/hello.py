import gi
gi.require_version("Gtk", "3.0")
from gi.repository import Gtk, GLib

w = Gtk.Window(title="gowt linux stand")
w.set_default_size(320, 120)
w.add(Gtk.Button(label="hello gtk3"))
w.connect("destroy", Gtk.main_quit)
w.show_all()
GLib.timeout_add_seconds(3, Gtk.main_quit)
Gtk.main()
