{{ define "itemheader" }}
          <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
            <h1 class="h2">创意管理</h1>
            <div class="btn-toolbar mb-2 mb-md-0">

<button type="button" class="btn btn-sm btn-outline-success and-all-other-classes"> 
  <a href="item?action=startnew&campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery}}" style="color:inherit"> 新建创意 </a>
</button>

            </div>
          </div>
{{ end }}
