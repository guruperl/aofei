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
          labels: ["周日", "周一", "周二", "周三", "周四", "周五", "周六"],
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

	<h2>创意报表</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>创意</th>
<th>花费</th>
<th>曝光次数</th>
<th>点击次数</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsAdvItem}}{{range .}}
<tr>
<td>{{.item_name}}</td>
<td>{{.spend}}</td>
<td>{{.imp}}</td>
<td>{{.cli}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

	<h2>活动报表</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>活动</th>
<th>花费</th>
<th>曝光次数</th>
<th>点击次数</th>
</tr></thead>
<tbody>{{with .Other.ledger_topicsAdvPub}}{{range .}}
<tr>
<td>{{.company}}</td>
<td>{{.spend}}</td>
<td>{{.imp}}</td>
<td>{{.cli}}</td>
</tr>{{end}}{{end}}
</tbody>
	</table>
	</div>

{{ template "footer" }}
