package fixture;

public enum Level {
	LOW, MID, HIGH;

	Level next() { return values()[(ordinal() + 1) % values().length]; }
}
