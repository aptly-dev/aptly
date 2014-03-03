set output 'mem.png'
set term png
set key box left
set xlabel "Time (msec)"
set ylabel "Mem (MB)"
plot "mem.dat" using 1:($2/1e6) title 'HeapSys' with lines, "mem.dat" using 1:($3/1e6) title 'HeapAlloc' with lines, "mem.dat" using 1:($4/1e6) title 'HeapIdle' with lines