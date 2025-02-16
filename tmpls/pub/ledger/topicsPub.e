{{ template "header" .}}
{{ template "ledgerheader" .}}

          <canvas class="my-4" id="pubChart" width="900" height="380"></canvas>

    <!-- Graphs -->
    <script src="https://cdnjs.cloudflare.com/ajax/libs/Chart.js/2.7.1/Chart.min.js"></script>
    <script>
      var ctx = document.getElementById("pubChart");
      var pubChart = new Chart(ctx, {
        type: 'line',
        data: {
          labels: ["Sunday", "Monday", "Tuesday", "Wednesday", "Thursday", "Friday", "Saturday"],
          datasets: [{
            data: [15339, 21345, 18483, 24003, 23489, 24092, 12034],
            lineTension: 0,
            backgroundColor: 'transparent',
            borderColor: '#007bff',
            borderWidth: 4,
            pointBackgroundColor: '#007bff'
          }]
        },
        options: {
          scales: { yAxes: [{ ticks: { beginAtZero: false } }] },
          legend: { display: false, }
        }
      });
    </script>

	<h2>Income by Slot</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Income</th>
<th>Impressions</th>
<th>Clicks</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsPubSlot}}{{range .}}
<tr>
<td>{{.slot_name}}</td>
<td>{{.income}}</td>
<td>{{.imp}}</td>
<td>{{.cli}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

	<h2>Income by Advertiser</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>Name</th>
<th>Income</th>
<th>Impressions</th>
<th>Clicks</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsPubAdv}}{{range .}}
<tr>
<td>{{.company}}</td>
<td>{{.income}}</td>
<td>{{.imp}}</td>
<td>{{.cli}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

{{ template "footer" }}
