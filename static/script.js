var events = {};

events.listeners = {};

events.publish = function (topic, args) {
    var listeners = events.listeners[topic];

    if (!listeners) {
        return;
    }

    for (var i = 0; i < listeners.length; i++) {
        listeners[i].apply(window, args);
    }
};

events.subscribe = function (topic, listener) {
    if (!events.listeners[topic]) {
        events.listeners[topic] = [];
    }

    events.listeners[topic].push(listener);
};

function ajax(method, path, successCallback, errorCallback) {
    var req = new XMLHttpRequest();

    req.onreadystatechange = function() {
        if (req.readyState == XMLHttpRequest.DONE) {
            if (req.status == 200) {
                successCallback(req.responseText);
            } else {
                errorCallback(req.status);
            }
        }
    }

    req.open(method, path, true);
    req.send();
}

// end of library code (TODO separate)

events.subscribe('progressUpdated', function (uuid, progress) {
    // update percentage
    var e = document.getElementById("done-" + uuid);
    e.style.width = progress.PctDone + "%";
});

events.subscribe('progressUpdated', function (uuid, progress) {
    // update "done / todo"
    var e = document.getElementById("pct-done-" + uuid);
    e.title = progress.Done + " / " + progress.Todo;
});

events.subscribe('progressUpdated', function () {
    // update weekly done
    var eDone = document.getElementById("this-week-done"),
        eTotal = document.getElementById("this-week-total"),
        weekDone = parseInt(eDone.innerHTML, 10),
        weekTotal = parseInt(eTotal.innerHTML, 10);

    weekDone += 1;

    eDone.innerHTML = weekDone;

    // update weekly percentage
    var e = document.getElementById("done-week");
    e.style.width = (weekDone / weekTotal * 100) + "%";
});

events.subscribe('progressUpdated', function (uuid, progress) {
    // update habit done
    var e = document.getElementById("points-done-" + uuid);
    e.innerHTML = progress.Done;
});

function updateActivityProgress(uuid) {
    ajax("POST", "/update/" + uuid, function (body) {
        var progress = document.getElementById("done-" + uuid);
        progress.style.width = body + "%";
    }, function (statusCode, body) {
        console.log("fail", statusCode, body);
    });

    return false;
}

function updateHabitProgress(uuid) {
    ajax("POST", "/habits/" + uuid, function (response) {
        events.publish('progressUpdated', [uuid, JSON.parse(response)]);
    }, function (statusCode, body) {
        console.log("fail", statusCode, body);
    });

    return false;
}
