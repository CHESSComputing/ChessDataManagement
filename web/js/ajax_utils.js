function ajaxCheckPid(base, method, input, inst, pid, view, interval) {
    // base is a URL base, e.g. https://cmsweb.cern.ch
    // method is request method, e.g. /request
    // pid is Query qhash
    // status request interval in seconds
    var limit = 10000; // in miliseconds
    var increment = 4000 // in milliseconds
    var wait  = parseInt(interval);
    if (wait+increment < limit) {
        wait  = wait+increment;
    } else if (wait==limit) {
        wait  = 2000; // initial time in msec (5 sec)
    } else { wait = limit; }
    new Ajax.Updater('response', base+'/'+method,
    { method: 'get' ,
        parameters : {'pid': pid, 'input': input, 'ajax': 1, 'instance': inst, 'view': view},
      onException: function() {return;},
      onComplete : function() {
//        if (url.indexOf('view=xml') != -1 ||
//            url.indexOf('view=json') != -1 ||
//            url.indexOf('view=plain') != -1) return;
          if(view == "plain") {
              location.reload(); // reload page
          }
          return
      },
      onSuccess : function(transport) {
        var sec = wait/1000;
        var msg = ', next check in '+sec.toString()+' sec, please wait..., <a href="/das/">stop</a> request';
        // look at transport body and match its content,
        // if check_pid still working on request, call again, otherwise
        // reload the request page
        if (transport.responseText.match(/request PID/)) {
            transport.responseText += msg;
            setTimeout('ajaxCheckPid("'+base+'","'+method+'","'+input+'","'+inst+'","'+pid+'","'+view+'","'+wait+'")', wait);
        } else {
            if(view == "plain") {
                location.reload(); // reload page
            }
            return;
        }
      }
    });
}

// workaround/bug-fix in prototype to make same-origin ajax easily
Ajax.Responders.register({
  onCreate: function(response) {
    // TODO: isSameOrigin() seem to fail at least for localhost in Chrome
    if (false && response.request.isSameOrigin())
      return;

    var t = response.transport;
    t.setRequestHeader = t.setRequestHeader.wrap(function(original, k, v) {

      if (/^(accept|accept-language|content-language)$/i.test(k))
        return original(k, v);
      if (/^content-type$/i.test(k) &&
          /^(application\/x-www-form-urlencoded|multipart\/form-data|text\/plain)(;.+)?$/i.test(v))
        return original(k, v);
      //return;
    });
  }
});
