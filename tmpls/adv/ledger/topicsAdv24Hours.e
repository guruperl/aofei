{{ template "header" .}}
{{ template "ledgerheader" .}}

	<canvas class="my-4" id="advChart" width="900" height="380"></canvas>

    <!-- Graphs -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.7.1/Chart.min.js"></script>
    <script>
      var ctx = document.getElementById("advChart");
      var advChart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: [{{range $index, $item := .Lists}}{{if $index}},{{end}}"{{$item.hours}}"{{end}}],
          datasets: [{
			yAxisID: 'y1',
			label: "Impressions",
            data: [{{range $index, $item := .Lists}}{{if $index}},{{end}}{{$item.imps}}{{end}}],
            backgroundColor: '#007bff',
            borderColor: '#007bff',
			fill: false
          },{
			yAxisID: 'y2',
            label: "Clicks",
            data: [{{range $index, $item := .Lists}}{{if $index}},{{end}}{{$item.clis}}{{end}}],
            backgroundColor: '#ff7b00',
            borderColor: '#ff7b00',
            fill: false
          },{
            yAxisID: 'y2',
            label: "Income",
            data: [{{range $index, $item := .Lists}}{{if $index}},{{end}}{{$item.spend}}{{end}}],
            backgroundColor: '#ff007b',
            borderColor: '#ff007b',
            fill: false
          }]
        },
        options: {
			responsive: true,
            scales: { yAxes: [{
						ticks: { beginAtZero: false },
						position: 'left',
						id: 'y1' },{
						gridLines: { drawOnChartArea: false },
						ticks: { beginAtZero: false },
						position: 'right',
						id: 'y2' }]
					},
            legend: { display: true }
        }
      });
    </script>

	<h3>Performance by Top Items</h3>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Spendings</th>
<th>Impressions</th>
<th>Clicks</th>
<th>CPM</th>
<th>CPC</th>
<th>CTR</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsAdvTopItems}}{{range .}}
<tr>
<td>{{.item_name}}</td>
<td>{{.spend}}</td>
<td>{{.imps}}</td>
<td>{{.clis}}</td>
<td>{{.cpm}}</td>
<td>{{.cpc}}</td>
<td>{{.ctr}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

	
<p></p>

    <h3>Performance by Top Slots</h3>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Spendings</th>
<th>Impressions</th>
<th>Clicks</th>
<th>CPM</th>
<th>CPC</th>
<th>CTR</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsAdvTopSlots}}{{range .}}
<tr>
<td>{{.slot_name}}</td>
<td>{{.spend}}</td>
<td>{{.imps}}</td>
<td>{{.clis}}</td>
<td>{{.cpm}}</td>
<td>{{.cpc}}</td>
<td>{{.ctr}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

{{ template "footer" }}
