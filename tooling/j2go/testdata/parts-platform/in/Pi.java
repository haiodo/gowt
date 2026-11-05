package fixture;

public class Pi {
	int handle;

	public Pi(int handle) {
		this.handle = handle;
	}

	public int getHandle() {
		return handle;
	}

	public void setHandle(int h) {
		this.handle = h;
	}

	public boolean isValid() {
		return handle != 0;
	}

	public void reset() {
		handle = 0;
	}
}
