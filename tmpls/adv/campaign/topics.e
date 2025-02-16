{{ template "header" .}}
{{ template "campaignheader" .}}

<h3>Current Campaigns</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>Name</th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                  <th></th>
                </tr>
              </thead>
              <tbody>{{ with .Lists }}{{ range . }}
<tr>
{{$small := print "campaign_id=" .campaign_id "&campaign_md5=" .campaign_md5 "&campaign_name=" (.campaign_name | urlquery)}}
<td>{{.campaign_name}}</td>
<td><a href="item?action=topics&{{$small}}">Item</a></td>
<td><a href="balance?action=topics&{{$small}}&entitytype_id=41">Finance</a></td>
<td><a href="targetname?action=topics&{{$small}}">Audience</a></td>
<td><a href="chac?action=topics&{{$small}}&entitytype_id=41">Channels</a></td>
<td><a href="ac?action=topics&{{$small}}&entitytype_id=41">BW</a></td>
<td><a href="campaign?action=edit&campaign_id={{.campaign_id}}">Edit</a></td>
<td><a onClick="return (confirm('Do you want to remove your site {{.campaign_name}}?')) ? true : false;" href="campaign?action=delete&campaign_id={{.campaign_id}}">Del</a></td>
</tr>
{{end}}{{end}}</tobdy>
</table>
</div>

{{ template "footer" }}
