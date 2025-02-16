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

	<h2>收入报表：广告位</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>广告位名称</th>
<th>收入</th>
<th>曝光量</th>
<th>点击量</th>
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

	<h2>收入报表：广告主</h2>
	<div class="table-responsive">
	<table class="table table-striped table-sm">
<thead><tr>
<th>广告主名称</th>
<th>收入</th>
<th>曝光量</th>
<th>点击量</th>
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
