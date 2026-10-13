package fixture;

public class Rec1 {
	record Point(int x, int y) {}
	int sum() {
		Point p = new Point(1, 2);
		return p.x() + p.y();
	}
}
