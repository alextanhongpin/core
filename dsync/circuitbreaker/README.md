func do(fn) {
    ok := init()
    if !ok {
        return unavailable
    }
    res, err := fn()
    update(err != nil)
    if err != nil {
        return
    }
}

func update(success) {
    status = getStatus()
    switch status {
    case "closed":
        return closed(success)
    case "opened":
        panic("not allowed")
    case "half-opened":
        return halfOpen(success)
    }
}

func init() {
    status = getStatus()
    switch status {
    case "closed":
        return allow
    case "opened":
        return open()
    case "half-opened":
        return allow
    }
}

func close(success) {
    if success {
        return ok
    }

    increment failure counter
    if failure threshold exceeded {
        transition(open)
    }
    return err
}

func halfOpen(success) {
    if success then
        increment success counter
        if success count threshold reached
            transition(closed)
        end
        return result

    transition(open)
    return failure
}

func open() {
    if timeout timer expired
        transition(half open)
        return allow
    end
    return failure
}

func transition(status) {
    setStatus(status)
    switch status {
    case "half-opened":
        onHalfOpen()
    case "open":
        onOpen()
    case "closed":
        onClosed()
    }
}


func onClosed() {
    reset failure counter
}

func onOpened() {
    start timeout timer
}

func onHalfOpen() {
    reset success counter
}
