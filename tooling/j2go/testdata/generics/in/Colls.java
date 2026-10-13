package fixture;

import java.util.ArrayList;
import java.util.HashMap;
import java.util.List;
import java.util.Map;

public class Colls {
	List<String> names = new ArrayList<>();
	Map<String, Integer> counts = new HashMap<>();
	void add(String n) {
		names.add(n);
		counts.put(n, counts.getOrDefault(n, 0) + 1);
	}
	int total() {
		int t = 0;
		for (String n : names) t += counts.get(n);
		for (Map.Entry<String, Integer> e : counts.entrySet()) t += e.getValue();
		return t;
	}
}
