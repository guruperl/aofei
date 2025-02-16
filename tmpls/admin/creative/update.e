<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Creative Updated.</title>        
  </head>
  <body>
    [% SET u = update.0 %]
    <script>
      var pageURL = "/go.fcgi/admin/e/creative?action=topics&itemid=[% GET u.itemid %]"

      alert( "Creative updated.\nYou will be redirected back to the admin page." )
      window.location = pageURL
    </script>  
  </body>  
</html>
