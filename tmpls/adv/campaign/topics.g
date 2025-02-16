{{ template "header" .}}
{{ template "campaignheader" .}}

<h3>推广活动</h3>
<div class="table-responsive">
<table class="table table-striped table-sm">
              <thead>
                <tr>
                  <th>活动名称</th>
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
<td><a href="item?action=topics&{{$small}}">创意</a></td>
<td><a href="balance?action=topics&{{$small}}&entitytype_id=41">总预算</a></td>
<td><a href="targetname?action=topics&{{$small}}">标签定向</a></td>
<td><a href="chac?action=topics&{{$small}}&entitytype_id=41">行业定向</a></td>
<td><a href="ac?action=topics&{{$small}}&entitytype_id=41">网站黑白名单</a></td>
<td><a href="campaign?action=edit&campaign_id={{.campaign_id}}">编辑活动</a></td>
<td><a onClick="return (confirm('您确定要删除活动（{{.campaign_name}}）吗?')) ? true : false;" href="campaign?action=delete&campaign_id={{.campaign_id}}">删除活动</a></td>
</tr>
{{end}}{{end}}</tobdy>
</table>
</div>

{{ template "footer" }}
