<!DOCTYPE html PUBLIC "-//W3C//DTD XHTML 1.1//EN" "http://www.w3.org/TR/xhtml11/DTD/xhtml11.dtd">
<html xmlns="http://www.w3.org/1999/xhtml">
  <head>
    <title>Item Updated.</title>
    <script type="text/javascript" src="/js/jquery-1.4.2.min.js"></script>
  </head>
  <body>    
    [% SET u = update.0 %]
    <script>
      var pageURL = "/go.fcgi/admin/e/item?action=topics&campaignid=[% GET u.campaignid %]"
    
      alert( "Item updated.\nYou will be redirected back to the admin page." )
      window.location = pageURL
    </script>  
  </body>  
</html>
