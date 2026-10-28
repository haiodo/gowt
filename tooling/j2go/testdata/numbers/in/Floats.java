package fixture;

public class Floats {
	double[] rgb() { return new double[] {236/255f, 235/255F, 1}; }
	double half() { return 1/2d; }
	double third() { return 1/3D * 3; }
	float ratio(int a, float b) { return a / b; }
	float scale(int a) { return a / 2f; }
	int truncated() { return 1 / 2; }
	float exact() { return 1.5f / 3 + 2e1f; }
}
