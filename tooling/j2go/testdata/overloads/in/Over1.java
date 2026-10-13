package fixture;

public class Over1 {
	void put(int v) {}
	void put(String v) {}
	void put(int a, int b) {}
	void put(Object o) {}
	void use() {
		put(1);
		put("s");
		put(1, 2);
		put(new Object());
	}
}
