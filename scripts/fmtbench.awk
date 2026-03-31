{
    name = $1
    cpus = "1"
    if (match(name, /-[0-9]+$/)) {
        cpus = substr(name, RSTART+1)
        name = substr(name, 10, RSTART-10)
    } else {
        name = substr(name, 10)
    }

    nsop  = $3
    bop   = 0
    alloc = 0

    for (i = 1; i <= NF; i++) {
        if ($i == "B/op"      && i > 1) bop   = $(i-1)
        if ($i == "allocs/op" && i > 1) alloc = $(i-1)
    }

    if (nsop > 0 && $4 == "ns/op") {
        ops = 1e9 / nsop
        if      (ops >= 1e9) opsstr = sprintf("%.2fB ops/s", ops/1e9)
        else if (ops >= 1e6) opsstr = sprintf("%.2fM ops/s", ops/1e6)
        else if (ops >= 1e3) opsstr = sprintf("%.2fK ops/s", ops/1e3)
        else                 opsstr = sprintf("%.0f  ops/s", ops)
    } else {
        opsstr = "       —     "
    }

    if (nsop >= 1000) latstr = sprintf("%.2f µs/op", nsop/1000)
    else              latstr = sprintf("%.2f ns/op", nsop)

    if (alloc == 0) allocstr = "zero-alloc"
    else            allocstr = sprintf("%d allocs/op  %d B/op", alloc, bop)

    cpulabel = sprintf("[%s core%s]", cpus, (cpus == "1" ? " " : "s"))

    printf "  %-22s %-8s  %13s   %10s   %s\n",
        tolower(name), cpulabel, opsstr, latstr, allocstr

    fflush()
}