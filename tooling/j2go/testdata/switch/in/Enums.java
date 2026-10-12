package fixture;

public class Enums {
	enum Color { RED, GREEN, BLUE }
	int code(Color c) {
		switch (c) {
			case RED: return 1;
			case GREEN: return 2;
			default: return 3;
		}
	}
}
