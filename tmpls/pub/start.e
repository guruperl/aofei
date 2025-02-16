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
      <a class="navbar-brand col-sm-3 col-md-2 mr-0" href="#">Publishers</a>
      <input class="form-control form-control-dark w-100" type="text" placeholder="Search" aria-label="Search">
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
                <a class="nav-link{{ if eq .Other.Component `ledger` }} active{{end}}" href="ledger?action=topicsPub">
                  <span data-feather="home"></span>
                  Dashboard {{ if eq .Other.Component "ledger" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `site` }} active{{end}}" href="site?action=topics">
                  <span data-feather="file"></span>
                  Websites {{ if eq .Other.Component "site" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              {{ if eq .Other.Component `slot` }}<li class="nav-item">
                <a class="nav-link active" href="slot?action=topics&site_id={{index .ARGS.site_id 0}}&site_md5={{index .ARGS.site_md5 0}}&site_name={{index .ARGS.site_name 0 | urlquery }}">
                  <span data-feather="file"></span>
                  Slots of {{index .ARGS.site_name 0}} <span class="sr-only">(current)</span>
                </a>
              </li>{{ end }}
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `ac` }} active{{end}}" href="ac?action=topics&entitytype_id=3">
                  <span data-feather="shopping-cart"></span>
                  Access Control {{ if eq .Other.Component "ac" }}<span class="sr-only">(current)</span>{{ end }}
                </a>
              </li>
              <li class="nav-item">
                <a class="nav-link{{ if eq .Other.Component `pub` }} active{{end}}" href="pub?action=edit">
                  <span data-feather="users"></span>
                  Settings {{ if eq .Other.Component "pub" }}<span class="sr-only">(current)</span>{{ end }}
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
