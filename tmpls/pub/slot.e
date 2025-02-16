{{ define "slotheader" }}
          <div class="d-flex justify-content-between flex-wrap flex-md-nowrap align-items-center pb-2 mb-3 border-bottom">
            <h1 class="h2">{{index .ARGS.site_name 0}}</h1>
            <div class="btn-toolbar mb-2 mb-md-0">

<button type="button" class="btn btn-sm btn-outline-success and-all-other-classes"> 
  <a href="slot?action=startnew&&site_id={{index .ARGS.site_id 0}}&site_md5={{index .ARGS.site_md5 0}}&site_name={{index .ARGS.site_name 0 | urlquery}}" style="color:inherit"> New Slot </a>
</button>

            </div>
          </div>
{{ end }}
