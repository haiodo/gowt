package fixture;

public class Dollar1 {
	private int $id;
	String this$name = "a$b";
	static int $count = 1;
	static final String $NAME = "cost $5";

	public int get$id() { return $id; }
	static int access$000(Dollar1 o) { return o.$id; }

	public boolean equals(Object o) {
		if (!(o instanceof Dollar1)) return false;
		Dollar1 other = (Dollar1) o;
		final Object this$id = this.get$id();
		final Object other$id = other.get$id();
		if (this$id == null ? other$id != null : !this$id.equals(other$id)) return false;
		return this.this$name.equals(other.this$name);
	}

	public int hashCode() {
		final int PRIME = 59;
		int result = 1;
		final Object $id = this.get$id();
		result = result * PRIME + ($id == null ? 43 : $id.hashCode());
		for (int $i = 0; $i < 3; $i++) result += $i;
		return result + $count;
	}

	static class In$ner {
		int v$;
		In$ner(int v$) { this.v$ = v$; }
	}

	int use(int p$1) {
		In$ner n = new In$ner(p$1);
		return n.v$ + access$000(this);
	}
}
