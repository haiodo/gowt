package fixture;

public enum Shape implements Named {
	CIRCLE {
		public String label() { return "round"; }
	},
	SQUARE {
		public String label() { return "boxy"; }
	};
}
