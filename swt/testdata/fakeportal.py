# A Settings portal for TestFollowSystemThemeAsync: slow to start, and only the v1 Read method
# (ReadOne fails with UnknownMethod), answering dark and a red accent.
import time
import gi
gi.require_version("Gio", "2.0")
from gi.repository import Gio, GLib

time.sleep(1)
XML = """<node><interface name="org.freedesktop.portal.Settings">
<method name="Read"><arg type="s" direction="in"/><arg type="s" direction="in"/><arg type="v" direction="out"/></method>
<signal name="SettingChanged"><arg type="s"/><arg type="s"/><arg type="v"/></signal>
</interface></node>"""
node = Gio.DBusNodeInfo.new_for_xml(XML)


def call(conn, sender, path, iface, method, params, inv):
    ns, key = params.unpack()
    if ns == "org.freedesktop.appearance" and key == "color-scheme":
        value = GLib.Variant("u", 1)
    elif ns == "org.freedesktop.appearance" and key == "accent-color":
        value = GLib.Variant("(ddd)", (0.9, 0.2, 0.2))
    else:
        inv.return_dbus_error("org.freedesktop.portal.Error.NotFound", "not found")
        return
    inv.return_value(GLib.Variant("(v)", (GLib.Variant("v", value),)))


def bus_acquired(conn, name):
    conn.register_object("/org/freedesktop/portal/desktop", node.interfaces[0], call, None, None)


Gio.bus_own_name(Gio.BusType.STARTER, "org.freedesktop.portal.Desktop", 0, bus_acquired, None, None)
GLib.MainLoop().run()
