package api

import (
	"context"

	"github.com/99designs/gqlgen/graphql"
	"github.com/99designs/gqlgen/graphql/errcode"
	"github.com/99designs/gqlgen/graphql/executor"
	"github.com/99designs/gqlgen/graphql/handler/extension"
	"github.com/99designs/gqlgen/graphql/handler/lru"
	"github.com/valyala/fasthttp"
	"github.com/vektah/gqlparser/v2/gqlerror"
	"github.com/viderstv/api/src/api/helpers"
)

type Wrapper struct {
	exec *executor.Executor
}

func (s *Wrapper) SetErrorPresenter(f graphql.ErrorPresenterFunc) {
	s.exec.SetErrorPresenter(f)
}

func (s *Wrapper) SetRecoverFunc(f graphql.RecoverFunc) {
	s.exec.SetRecoverFunc(f)
}

func (s *Wrapper) SetQueryCache(cache graphql.Cache) {
	s.exec.SetQueryCache(cache)
}

func (s *Wrapper) Use(extension graphql.HandlerExtension) {
	s.exec.Use(extension)
}

// AroundFields is a convenience method for creating an extension that only implements field middleware
func (s *Wrapper) AroundFields(f graphql.FieldMiddleware) {
	s.exec.AroundFields(f)
}

// AroundOperations is a convenience method for creating an extension that only implements operation middleware
func (s *Wrapper) AroundOperations(f graphql.OperationMiddleware) {
	s.exec.AroundOperations(f)
}

// AroundResponses is a convenience method for creating an extension that only implements response middleware
func (s *Wrapper) AroundResponses(f graphql.ResponseMiddleware) {
	s.exec.AroundResponses(f)
}

func NewDefaultWrapper(es graphql.ExecutableSchema) *Wrapper {
	srv := NewWrapper(es)

	srv.SetQueryCache(lru.New(1000))

	srv.Use(extension.Introspection{})
	srv.Use(extension.AutomaticPersistedQuery{
		Cache: lru.New(100),
	})

	return srv
}

func NewWrapper(es graphql.ExecutableSchema) *Wrapper {
	return &Wrapper{
		exec: executor.New(es),
	}
}

func statusFor(errs gqlerror.List) int {
	switch errcode.GetErrorKind(errs) {
	case errcode.KindProtocol:
		return fasthttp.StatusUnprocessableEntity
	case errcode.KindUser:
		if len(errs) == 0 {
			return fasthttp.StatusOK
		}
		if len(errs) == 1 {
			if errs[0].Message == helpers.ErrInternalServerError.Error() {
				return fasthttp.StatusInternalServerError
			}
		}
		return fasthttp.StatusBadRequest
	default:
		return fasthttp.StatusOK
	}
}

func ProcessExecution(params *graphql.RawParams, exec graphql.GraphExecutor, baseContext context.Context) ReturnSignal {
	start := graphql.Now()
	params.ReadTime = graphql.TraceTiming{Start: start, End: graphql.Now()}

	response, listOfErrors := exec.CreateOperationContext(baseContext, params)
	if listOfErrors != nil {
		resp := exec.DispatchError(graphql.WithOperationContext(baseContext, response), listOfErrors)
		return ReturnSignal{
			Status:   statusFor(listOfErrors),
			Response: resp,
		}
	}
	responses, ctx := exec.DispatchOperation(baseContext, response)
	resp := responses(ctx)

	return ReturnSignal{
		Status:   statusFor(resp.Errors),
		Response: resp,
	}
}

type Response struct {
	Error   string `json:"error,omitempty"`
	Message string `json:"message,omitempty"`
	Reason  string `json:"reason,omitempty"`
	ReturnSignal
}

type ReturnSignal struct {
	Status   int `json:"status,omitempty"`
	Response *graphql.Response
}

func (s *Wrapper) Process(ctx context.Context, params graphql.RawParams) (resp Response) {
	defer func() {
		if err := recover(); err != nil {
			err := s.exec.PresentRecoveredError(ctx, err)
			resp = Response{
				Message: "internal server error",
				ReturnSignal: ReturnSignal{
					Status:   fasthttp.StatusInternalServerError,
					Response: &graphql.Response{Errors: []*gqlerror.Error{err}},
				},
			}
			return
		}
	}()

	childContext := graphql.StartOperationTrace(ctx)
	output := ProcessExecution(&params, s.exec, childContext)
	return Response{
		ReturnSignal: output,
	}
}
