package fixture;

public class TryCatch {
	int parse(String s) {
		try {
			return Integer.parseInt(s);
		} catch (NumberFormatException e) {
			return -1;
		}
	}
	void multi(Object o) {
		try {
			o.toString();
		} catch (NullPointerException | IllegalStateException e) {
			System.out.println(e.getMessage());
		}
	}
}
