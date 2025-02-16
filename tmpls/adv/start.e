{{ define "header" }}
<!doctype html>
<html lang="en">
  <head>
    <meta charset="utf-8">
    <meta name="viewport" content="width=device-width, initial-scale=1, shrink-to-fit=no">
    <meta name="description" content="">
    <meta name="author" content="">
    <link rel="icon" href="/favicon.ico">

    <title>Kinet Publisher Management</title>

    <!-- Bootstrap core CSS -->
    <link href="/dist/css/bootstrap.min.css" rel="stylesheet">

    <!-- Custom styles for this template -->
    <link href="/dashboard.css" rel="stylesheet">
  </head>

  <body>
    <nav class="navbar navbar-dark sticky-top bg-dark flex-md-nowrap p-0">
      <a class="navbar-brand col-sm-3 col-md-2 mr-0" href="#">Advertiser</a>
    <div class="navbar-brand">
Welcome&nbsp; <em>{{index .ARGS.a_email 0}}</em> of <em>{{index .ARGS.a_company 0}}</em> ! Your ID&nbsp; <em>{{index .ARGS.adv_id 0}}</em>.
	</div>
      <ul class="navbar-nav px-3">
        <li class="nav-item text-nowrap">
          <a class="nav-link" href="logout">Sign out</a>
        </li>
      </ul>
    </nav>

    <div class="container-fluid">
      <div class="row">
        <nav class="col-md-2 d-none d-md-block bg-light sidebar">
          <div class="sidebar-sticky">
            <ul class="nav flex-column">
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `ledger` }} active{{end}}" href="ledger?action=topicsAdv24Hours">
                  <span data-feather="home"></span>
                  Dashboard {{ if eq .Other.Component "ledger" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `campaign` }} active{{end}}" href="campaign?action=topics">
                  <span data-feather="file"></span>
                  Campaigns {{ if eq .Other.Component "campaign" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              {{ if eq .Other.Component `item` }}<li class="nav-item">
                <a class="nav-link active" href="item?action=topics&campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery }}">
                  <span data-feather="file"></span>
                  Items of {{index .ARGS.campaign_name 0}} <span class="sr-only">(current)</span>
                </a>
              </li>{{ end }}
              {{ if eq .Other.Component `chac` }}<li class="nav-item">
                <a class="nav-link active" href="chac?action=topics&campaign_id={{index .ARGS.campaign_id 0}}&campaign_md5={{index .ARGS.campaign_md5 0}}&campaign_name={{index .ARGS.campaign_name 0 | urlquery }}&entitytype_id=41">
                  <span data-feather="file"></span>
                  Channels of {{index .ARGS.campaign_name 0}} <span class="sr-only">(current)</span>
                </a>
              </li>{{ end }}
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `ac` }} active{{end}}" href="ac?action=topics&entitytype_id=4">
                  <span data-feather="shopping-cart"></span>
                  Access Control {{ if eq .Other.Component "ac" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `adv` }} active{{end}}" href="adv?action=edit">
                  <span data-feather="users"></span>
                  Settings {{ if eq .Other.Component "adv" }}<span class="sr-only">(current)</span>{{ end }}
<li>
sfsdf
</li>
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `attrname` }} active{{end}}" href="attrname?action=topics">
                  <span data-feather="users"></span>
                  Custom Tag {{ if eq .Other.Component "attrname" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link" href="#">
                  <span data-feather="bar-chart-2"></span>
                  Reports
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link" href="#">
                  <span data-feather="layers"></span>
                  Integrations
                </a>
              </li>
            </ul>

          </div>
        </nav>

        <main role="main" class="col-md-9 ml-sm-auto col-lg-10 pt-3 px-4">

{{ end }}
