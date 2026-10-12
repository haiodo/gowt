package fixture;

public class Iface {
	interface Shape { double area(); default String label() { return "shape"; } }
	static class Sq implements Shape {
		double s;
		public double area() { return s * s; }
	}
	static class Circle implements Shape {
		double r;
		public double area() { return 3.14 * r * r; }
		@Override public String label() { return "circle"; }
	}
	static double total(Shape[] shapes) {
		double t = 0;
		for (Shape sh : shapes) t += sh.area();
		return t;
	}
}
