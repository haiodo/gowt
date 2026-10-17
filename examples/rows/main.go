// Command rows shows a table and a tree whose rows follow rx state: each tick changes a few
// elements, adds or drops one, and only those rows are touched; selection and expansion stay.
package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"slices"
	"time"

	"github.com/haiodo/gowt"
	"github.com/haiodo/gowt/rx"
)

// job is one table row. It is comparable: a row is filled again only when its job changed.
type job struct {
	id     int
	name   string
	state  string
	group  string
	cpuPct int
}

// entry is one tree node: a group, or a job inside it.
type entry struct {
	key, text, detail string
}

func main() {
	err := gowt.Run(func(app *gowt.App) {
		jobs := rx.NewFunc(startJobs(), nil)
		selected := rx.New("Select a row")
		groups := rx.ComputeFunc(func() []entry { return groupsOf(jobs.Get()) }, nil)
		members := func(e entry) []entry { return membersOf(jobs.Peek(), e) }

		var table *gowt.TableView
		w := app.Window("Rows")
		w.Add(
			gowt.Label("", gowt.BindText(selected)),
			gowt.HBox(gowt.Grow(),
				gowt.Table(gowt.Grow(), gowt.Size(360, 280), gowt.Ref(&table),
					gowt.Init(func(t *gowt.TableView) {
						t.Column("Job", 140)
						t.Column("State", 90)
						t.Column("CPU", 60, gowt.Right())
					}),
					gowt.Rows(jobs, func(j job) int { return j.id }, func(r *gowt.TableRow, j job) {
						r.SetTextAt(0, j.name)
						r.SetTextAt(1, j.state)
						r.SetTextAt(2, fmt.Sprint(j.cpuPct, "%"))
					}),
					gowt.OnSelect(func(i int) {
						if i >= 0 {
							selected.Set("Selected: " + table.RowAt(i).TextAt(0))
						}
					})),
				gowt.Tree(gowt.Grow(), gowt.Size(260, 280),
					gowt.Init(func(t *gowt.TreeView) {
						t.Column("Group", 160)
						t.Column("CPU", 60, gowt.Right())
					}),
					// children reads jobs untracked: groups already depends on it.
					gowt.Nodes(groups, func(e entry) string { return e.key }, members,
						func(n *gowt.Node, e entry, created bool) {
							n.SetText(e.text)
							n.SetTextAt(1, e.detail)
							if created {
								n.SetExpanded(true)
							}
						}))))

		app.Every(time.Second, func() { jobs.Set(tick(jobs.Peek())) })
		w.Show()
	})
	if err != nil {
		log.Fatal(err)
	}
}

var groupNames = []string{"build", "test", "deploy"}

func startJobs() []job {
	var js []job
	for i := range 12 {
		js = append(js, job{id: i, name: fmt.Sprint("job-", i), state: "running", group: groupNames[i%3], cpuPct: 10})
	}
	return js
}

// tick returns the next list: a few CPU values change, sometimes a job ends or a new one starts.
func tick(js []job) []job {
	next := slices.Clone(js)
	for range 3 {
		next[rand.IntN(len(next))].cpuPct = rand.IntN(100)
	}
	if len(next) > 6 && rand.IntN(3) == 0 {
		i := rand.IntN(len(next))
		next = slices.Delete(next, i, i+1)
	}
	if rand.IntN(2) == 0 {
		id := next[len(next)-1].id + 1
		next = append(next, job{id: id, name: fmt.Sprint("job-", id), state: "queued", group: groupNames[id%3]})
	}
	return next
}

func groupsOf(js []job) []entry {
	var out []entry
	for _, g := range groupNames {
		total, n := 0, 0
		for _, j := range js {
			if j.group == g {
				total += j.cpuPct
				n++
			}
		}
		out = append(out, entry{key: g, text: fmt.Sprintf("%s (%d)", g, n), detail: fmt.Sprint(total, "%")})
	}
	return out
}

func membersOf(js []job, group entry) []entry {
	var out []entry
	for _, j := range js {
		if j.group == group.key {
			out = append(out, entry{key: fmt.Sprint(j.id), text: j.name, detail: fmt.Sprint(j.cpuPct, "%")})
		}
	}
	return out
}
