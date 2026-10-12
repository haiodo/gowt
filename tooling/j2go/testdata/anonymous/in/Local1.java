package fixture;

public class Local1 {
	int run() {
		class Counter {
			int n;
			void inc() { n++; }
		}
		Counter c = new Counter();
		c.inc();
		c.inc();
		return c.n;
	}
}
