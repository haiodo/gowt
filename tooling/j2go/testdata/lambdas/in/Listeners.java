package fixture;

public class Listeners {
	interface Listener { void changed(String what, int old, int now); }
	Listener l;
	void set(Listener l) { this.l = l; }
	void wire() {
		set((what, old, now) -> {
			if (old != now) System.out.println(what);
		});
	}
}
