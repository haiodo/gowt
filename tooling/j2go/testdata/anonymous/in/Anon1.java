package fixture;

public class Anon1 {
	interface Cb { void call(int v); }
	int total;
	Cb make() {
		return new Cb() {
			public void call(int v) { total += v; }
		};
	}
	Runnable runner(final String msg) {
		return new Runnable() {
			public void run() { System.out.println(msg); }
		};
	}
}
