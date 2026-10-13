package fixture;

public @interface Marker {
	String value() default "";

	int[] ids() default {};

	Level level() default Level.LOW;
}
