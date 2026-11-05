package dev.gowt.j2go.emit;

import java.util.*;

/**
 * Declaration-aware output model: instead of one text blob per compilation unit, methods are
 * emitted as named chunks (owner Go type + method-name group + the exact body text, with the
 * generated helpers their emission pulled in attached). A {@link FilePlanner} packs chunks into
 * part files; the base file keeps types, fields, static and deferred init, shared helpers and
 * declarations that must stay together. Imports are computed per output file by the emitter.
 */
public final class OutputModel {

	private OutputModel() {}

	/** One method (constructor/instance/static) plus the declarations emission attached to it:
	 *  dispatch wrappers, hook wrappers, generated helpers and anonymous struct types. Overloads
	 *  share a group; constructors group as {@code New<Owner>}. */
	public static final class Chunk {
		public final String owner; // enclosing Go type name (nested types included)
		public final String group; // method name the part file is named after
		public final StringBuilder text = new StringBuilder();

		public Chunk(String owner, String group, String text) {
			this.owner = owner;
			this.group = group;
			this.text.append(text);
		}

		public int lines() {
			int n = 1;
			for (int i = 0; i < text.length(); i++) if (text.charAt(i) == '\n') n++;
			return n;
		}
	}

	/** Chunks packed into one output file. {@link #suffix} "" names the base file. Imports are
	 *  not part of the model: the emitter computes each output file's imports from the full unit
	 *  import set by textual reference, exactly as the single-file layout did. */
	public static final class Part {
		public final List<Chunk> chunks = new ArrayList<>();
		public String nameKey = ""; // unsanitized name basis, set at creation
		public String suffix = "";

		public Part(Collection<Chunk> chunks, String nameKey) {
			this.chunks.addAll(chunks);
			this.nameKey = nameKey;
		}

		public int lines() {
			int n = 0;
			for (Chunk c : chunks) n += c.lines();
			return n;
		}

		public String body() {
			StringBuilder b = new StringBuilder();
			for (Chunk c : chunks) b.append(c.text);
			return b.toString();
		}
	}

	/** One output file: suffix "" is the base file (the caller keeps its current name), otherwise
	 *  a method-named part file. */
	public record OutputFile(String suffix, String body, Set<String> imports) {}
}
