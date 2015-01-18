package project

import (
	"fmt"
	"net/http"

	restful "github.com/emicklei/go-restful"
	"github.com/facebookgo/stackerr"
	"github.com/sirupsen/logrus"

	"github.com/bearded-web/bearded/models/project"
	"github.com/bearded-web/bearded/pkg/filters"
	"github.com/bearded-web/bearded/pkg/manager"
	"github.com/bearded-web/bearded/pkg/pagination"
	"github.com/bearded-web/bearded/services"
)

const ParamId = "project-id"

type ProjectService struct {
	*services.BaseService
}

func New(base *services.BaseService) *ProjectService {
	return &ProjectService{
		BaseService: base,
	}
}

func addDefaults(r *restful.RouteBuilder) {
	r.Notes("Authorization required")
	r.Do(services.ReturnsE(
		http.StatusUnauthorized,
		http.StatusInternalServerError,
	))
}

// Fix for IntelijIdea inpsections. Cause it can't investigate anonymous method results =(
func (s *ProjectService) Manager() *manager.Manager {
	return s.BaseService.Manager()
}

func (s *ProjectService) Register(container *restful.Container) {
	ws := &restful.WebService{}
	ws.Path("/api/v1/projects")
	ws.Doc("Manage Projects")
	ws.Consumes(restful.MIME_JSON)
	ws.Produces(restful.MIME_JSON)
	ws.Filter(filters.AuthRequiredFilter())

	r := ws.GET("").To(s.list)
	r.Doc("list")
	r.Operation("list")
	r.Writes(project.ProjectList{})
	r.Do(services.Returns(http.StatusOK))
	addDefaults(r)
	ws.Route(r)

	r = ws.POST("").To(s.create)
	r.Doc("create")
	r.Operation("create")
	r.Writes(project.Project{})
	r.Reads(project.Project{})
	r.Do(services.Returns(http.StatusCreated))
	r.Do(services.ReturnsE(http.StatusConflict))
	addDefaults(r)
	ws.Route(r)

	r = ws.GET(fmt.Sprintf("{%s}", ParamId)).To(s.get)
	r.Doc("get")
	r.Operation("get")
	r.Param(ws.PathParameter(ParamId, ""))
	r.Writes(project.Project{})
	r.Do(services.Returns(
		http.StatusOK,
		http.StatusNotFound))
	r.Do(services.ReturnsE(http.StatusBadRequest))
	addDefaults(r)
	ws.Route(r)

	//	r = ws.PUT(fmt.Sprintf("{%s}", ParamId)).To(s.update)
	//	// docs
	//	r.Doc("update")
	//	r.Operation("update")
	//	r.Param(ws.PathParameter(ParamId, ""))
	//	r.Writes(project.Project{})
	//	r.Reads(project.Project{})
	//	r.Do(services.Returns(
	//		http.StatusOK,
	//		http.StatusNotFound))
	//	r.Do(services.ReturnsE(
	//		http.StatusBadRequest,
	//		http.StatusInternalServerError))
	//	ws.Route(r)

	container.Add(ws)
}

func (s *ProjectService) create(req *restful.Request, resp *restful.Response) {
	session := filters.GetSession(req)
	userId, _ := session.Get("userId")

	// TODO (m0sth8): Check permissions for the user, he is might be blocked or removed

	raw := &project.Project{}

	if err := req.ReadEntity(raw); err != nil {
		logrus.Error(stackerr.Wrap(err))
		resp.WriteServiceError(http.StatusBadRequest, services.WrongEntityErr)
		return
	}

	mgr := s.Manager()
	defer mgr.Close()

	raw.Owner = mgr.ToId(userId)
	obj, err := mgr.Projects.Create(raw)
	if err != nil {
		if mgr.IsDup(err) {
			resp.WriteServiceError(
				http.StatusConflict,
				services.NewError(services.CodeDuplicate, "project with this name and owner is existed"))
			return
		}
		logrus.Error(stackerr.Wrap(err))
		resp.WriteServiceError(http.StatusInternalServerError, services.DbErr)
		return
	}

	resp.WriteHeader(http.StatusCreated)
	resp.WriteEntity(obj)
}

func (s *ProjectService) list(_ *restful.Request, resp *restful.Response) {
	mgr := s.Manager()
	defer mgr.Close()

	results, count, err := mgr.Projects.All()
	if err != nil {
		logrus.Error(stackerr.Wrap(err))
		resp.WriteServiceError(http.StatusInternalServerError, services.DbErr)
		return
	}

	result := &project.ProjectList{
		Meta:    pagination.Meta{count, "", ""},
		Results: results,
	}
	resp.WriteEntity(result)
}

func (s *ProjectService) get(req *restful.Request, resp *restful.Response) {
	projectId := req.PathParameter(ParamId)
	if !s.IsId(projectId) {
		resp.WriteServiceError(http.StatusBadRequest, services.IdHexErr)
		return
	}

	mgr := s.Manager()
	defer mgr.Close()

	// TODO (m0sth8): check permissions for the user
	u, err := mgr.Projects.GetById(projectId)
	if err != nil {
		if mgr.IsNotFound(err) {
			resp.WriteErrorString(http.StatusNotFound, "Not found")
			return
		}
		logrus.Error(stackerr.Wrap(err))
		resp.WriteServiceError(http.StatusInternalServerError, services.DbErr)
		return
	}

	resp.WriteEntity(u)
}
